# Generated from the trusted shared PR-Agent runtime.
#!/usr/bin/env python
"""Trusted pilot entrypoint. Inputs are data; never shell or CLI fragments."""
import json
import os
import re
import sys
import tempfile
from pathlib import Path
from urllib.request import Request, urlopen

REPOSITORY = os.environ["PR_REPOSITORY"]
MODEL = "openrouter/moonshotai/kimi-k2.7-code"

# Embedded in the trusted Task by index.cjs; never loaded from a PR checkout.
import html
from types import SimpleNamespace


def review_policy():
    return json.loads(os.environ.get("PR_REVIEW_POLICIES", "{}")).get(REPOSITORY, {})


def review_coverage(reviewer):
    from pr_agent.algo.file_filter import filter_ignored

    provider = reviewer.git_provider
    paths = sorted({file if isinstance(file, str) else file.filename for file in provider.get_files()})
    files = [SimpleNamespace(filename=path) for path in paths]
    selected = {file.filename for file in filter_ignored(files)}
    analyzable = {file.filename for file in provider.get_diff_files() if file.patch}
    omitted = sorted(set(reviewer.remaining_files_list) | set(getattr(reviewer, "_chunked_remaining_files_list", [])))
    partial = sorted(set(reviewer.partial_files_list) | set(getattr(reviewer, "_chunked_partial_files_list", [])))
    failed = reviewer.review_failed_chunk_count
    if not isinstance(failed, int) or failed < 0:
        raise ValueError("Missing chunk coverage")
    if not (analyzable | set(omitted) | set(partial)) <= set(paths):
        raise ValueError("Coverage differs from the changed-file inventory")
    excluded = sorted(set(paths) - selected)
    other = sorted(selected - analyzable)
    expansions = getattr(provider, "_pr_agent_expanded_contracts", [])
    expanded_paths = {item["path"] for item in expansions}
    contracts = sorted((set(paths) & set(review_policy().get("unexpanded_contract_paths", []))) - expanded_paths)
    reviewed = sorted(analyzable - set(omitted) - set(partial)) if not failed else []
    return {
        "status": "partial" if omitted or partial or failed or other or contracts else "complete",
        "changed_files": getattr(provider, "_pr_agent_original_changed_files", len(paths)),
        "expanded_contracts": expansions,
        "expanded_contract_files": sorted({path for item in expansions for path in item["files"]}),
        "contract_expansion_failures": getattr(provider, "_pr_agent_contract_failures", []),
        "files_reviewed": reviewed,
        "files_excluded_generated": excluded,
        "files_not_analyzable": other,
        "files_not_reviewed": omitted,
        "files_partially_reviewed": partial,
        "unexpanded_contract_paths": contracts,
        "chunks": max(1, reviewer.review_chunk_count),
        "failed_chunks": failed,
        "reviewed_file_count_known": not bool(failed),
    }


def render_review_coverage(coverage):
    incomplete = coverage["status"] == "partial"
    heading = "⚠️ Partial PR-Agent review" if incomplete else "PR-Agent source review coverage"
    reviewed = str(len(coverage["files_reviewed"])) if coverage["reviewed_file_count_known"] else "unknown"
    lines = [f"## {heading}", "",
             f"Changed files: **{coverage['changed_files']}** · Fully reviewed patches: **{reviewed}** · "
             f"Chunks: **{coverage['chunks']}** · Failed chunks: **{coverage['failed_chunks']}**.", ""]
    if coverage.get("expanded_contract_files"):
        lines += [f"Expanded contract source files: **{len(coverage['expanded_contract_files'])}** "
                  "(additional to the parent repository's changed-file count).", ""]
    for expansion in coverage.get("expanded_contracts", []):
        url = f"https://github.com/{expansion['repository']}/compare/{expansion['base']}...{expansion['head']}"
        lines += [f"Expanded `{expansion['path']}` from its exact pinned [contract diff]({url}) "
                  f"into {len(expansion['files'])} source patches.", ""]
    if incomplete:
        lines += ["**The findings below apply only to the analyzed input. This is not a complete PR review.**", ""]
    for key, title in (
        ("files_excluded_generated", "Generated outputs intentionally excluded from AI review"),
        ("files_not_analyzable", "Files without an analyzable patch (provider filtering or unavailable diff)"),
        ("files_not_reviewed", "Files omitted by the token/call budget"),
        ("files_partially_reviewed", "Files with only part of the patch reviewed"),
        ("unexpanded_contract_paths", "Contract/submodule pointers whose underlying changes are not expanded"),
    ):
        paths = coverage[key]
        if paths:
            lines += [f"<details><summary>{title}: {len(paths)}</summary>", ""]
            lines += [f"- <code>{html.escape(path)}</code>" for path in paths[:50]]
            if len(paths) > 50:
                lines += [f"- … and {len(paths) - 50} more (full inventory in structured coverage receipts)."]
            lines += ["", "</details>", ""]
    for failure in coverage.get("contract_expansion_failures", []):
        lines += [f"Contract source expansion unavailable for <code>{html.escape(failure['path'])}</code>: "
                  f"<code>{html.escape(failure['reason'])}</code>.", ""]
    if coverage["files_excluded_generated"]:
        check = html.escape(review_policy().get("generation_check", "repository generation and build checks"))
        lines += [f"Validate excluded outputs separately in CI: <code>{check}</code>. "
                  "This advisory review does not attest that those checks passed.", ""]
    if coverage["unexpanded_contract_paths"]:
        lines += ["Review the corresponding contract/API PR as well; a submodule SHA diff does not include its source changes.", ""]
    return "\n".join(lines) + "\n---\n\n"


def install_coverage_reporting():
    from pr_agent.tools.pr_reviewer import PRReviewer

    original = PRReviewer._prepare_pr_review
    reports = []

    def prepare(self, *args, **kwargs):
        text = original(self, *args, **kwargs)
        if not isinstance(text, str) or not text.strip():
            raise ValueError("Missing rendered review")
        coverage = review_coverage(self)
        self._structured_review_data["coverage"] = coverage
        reports.append(coverage)
        return render_review_coverage(coverage) + text

    PRReviewer._prepare_pr_review = prepare
    return original, reports


# Host-owned mappings expand gitlinks as data, never as checked-out executable code.
from urllib.parse import quote


class ContractExpansionUnavailable(RuntimeError):
    pass


def contract_fetch_json(url):
    request = Request(url, headers={
        "Authorization": "Bearer " + (os.environ.get("PR_CONTRACT_READ_TOKEN") if "/compare/" in url and os.environ.get("PR_CONTRACT_READ_TOKEN") else os.environ["GITHUB__USER_TOKEN"]),
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
    })
    with urlopen(request, timeout=15) as response:
        payload = response.read(2 * 1024 * 1024 + 1)
    if len(payload) > 2 * 1024 * 1024:
        raise ContractExpansionUnavailable("response_limit")
    return json.loads(payload)


def contract_patch_complete(file):
    patch = file.get("patch")
    if not isinstance(patch, str) or not patch:
        return False
    added = sum(line.startswith("+") for line in patch.splitlines())
    removed = sum(line.startswith("-") for line in patch.splitlines())
    return added == file.get("additions") and removed == file.get("deletions")


def expand_contract(provider, file, repository):
    from pr_agent.algo.types import EDIT_TYPE, FilePatchInfo
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
        raise ContractExpansionUnavailable("invalid_repository_policy")
    old = re.findall(r"(?m)^-Subproject commit ([0-9a-f]{40})$", file.patch or "")
    new = re.findall(r"(?m)^\+Subproject commit ([0-9a-f]{40})$", file.patch or "")
    if len(old) != 1 or len(new) != 1:
        raise ContractExpansionUnavailable("unavailable_gitlink_range")
    old, new = old[0], new[0]
    head = provider.pr.head.sha
    if not re.fullmatch(r"[0-9a-f]{40}", head):
        raise ContractExpansionUnavailable("invalid_pr_revision")
    metadata = contract_fetch_json(
        f"https://api.github.com/repos/{REPOSITORY}/contents/{quote(file.filename, safe='')}?ref={head}")
    if metadata.get("sha") != new or metadata.get("submodule_git_url") != f"https://github.com/{repository}.git":
        raise ContractExpansionUnavailable("gitlink_identity_mismatch")
    comparison = contract_fetch_json(
        f"https://api.github.com/repos/{repository}/compare/{old}...{new}?per_page=1")
    # GitHub compares from merge-base, not necessarily the requested old pin.
    if comparison.get("status") != "ahead" or comparison.get("merge_base_commit", {}).get("sha") != old:
        raise ContractExpansionUnavailable("non_ancestor_range")
    files = comparison.get("files")
    # Compare API returns at most 300 files; stay below that ambiguous boundary.
    if not isinstance(files, list) or not files or len(files) > 100:
        raise ContractExpansionUnavailable("file_inventory_limit")
    expanded, names = [], set()
    types = {"added": EDIT_TYPE.ADDED, "modified": EDIT_TYPE.MODIFIED,
             "removed": EDIT_TYPE.DELETED, "renamed": EDIT_TYPE.RENAMED}
    for item in files:
        path = item.get("filename", "")
        if (not path or path.startswith("/") or any(part in ("", ".", "..") for part in path.split("/"))
                or any(ord(character) < 32 for character in path) or path in names):
            raise ContractExpansionUnavailable("invalid_file_inventory")
        if item.get("status") not in types or not contract_patch_complete(item):
            raise ContractExpansionUnavailable("unavailable_or_incomplete_patch")
        names.add(path)
        expanded.append(FilePatchInfo("", "", item["patch"], f"{file.filename}/{path}",
            edit_type=types[item["status"]],
            old_filename=f"{file.filename}/{item['previous_filename']}" if item.get("previous_filename") else None,
            num_plus_lines=item["additions"], num_minus_lines=item["deletions"], head_file_is_complete=False))
    return expanded, {"path": file.filename, "repository": repository, "base": old, "head": new,
                      "files": [item.filename for item in expanded]}


def install_contract_expansion():
    from pr_agent.git_providers.github_provider import GithubProvider
    original_files = getattr(GithubProvider, "get_files", None)
    original_diffs = getattr(GithubProvider, "get_diff_files", None)
    if original_files is None or original_diffs is None:
        return original_files, original_diffs

    def inventory(self):
        if hasattr(self, "_pr_agent_contract_inventory"):
            return self._pr_agent_contract_inventory
        self._pr_agent_contract_building = True
        try:
            paths, diffs = list(original_files(self)), list(original_diffs(self))
            original_count = len(paths)
            expanded, failures = [], []
            mapping = review_policy().get("contract_repositories", {})
            for file in diffs:
                repository = mapping.get(file.filename)
                if not repository:
                    continue
                try:
                    additions, receipt = expand_contract(self, file, repository)
                except Exception as error:
                    # Only fixed reason codes reach receipts. Provider errors may contain private data.
                    reason = str(error) if isinstance(error, ContractExpansionUnavailable) else "provider_read_failed"
                    failures.append({"path": file.filename, "reason": reason})
                else:
                    paths.extend(item.filename for item in additions)
                    expanded.extend(additions)
                    self._pr_agent_expanded_contracts = getattr(self, "_pr_agent_expanded_contracts", []) + [receipt]
            self._pr_agent_original_changed_files = original_count
            self._pr_agent_contract_failures = failures
            self._pr_agent_contract_inventory = paths, diffs + expanded
            return self._pr_agent_contract_inventory
        finally:
            self._pr_agent_contract_building = False

    def get_files(self):
        if getattr(self, "repo", None) != REPOSITORY or getattr(self, "_pr_agent_contract_building", False):
            return original_files(self)
        return inventory(self)[0]

    def get_diff_files(self):
        if getattr(self, "repo", None) != REPOSITORY or getattr(self, "_pr_agent_contract_building", False):
            return original_diffs(self)
        return inventory(self)[1]

    GithubProvider.get_files = get_files
    GithubProvider.get_diff_files = get_diff_files
    return original_files, original_diffs


class StalePullRequest(RuntimeError):
    pass

def fetch_pr(number):
    request = Request(
        f"https://api.github.com/repos/{REPOSITORY}/pulls/{number}",
        headers={
            "Authorization": f"Bearer {os.environ['GITHUB__USER_TOKEN']}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    with urlopen(request, timeout=30) as response:
        return json.load(response)

def eligible(pr, revision):
    return (
        pr.get("state") == "open"
        and not pr.get("draft", True)
        and pr.get("base", {}).get("ref") == "main"
        and pr.get("base", {}).get("repo", {}).get("full_name") == REPOSITORY
        and (pr.get("head", {}).get("repo") or {}).get("full_name") == REPOSITORY
        and pr.get("head", {}).get("sha") == revision
        and (os.environ.get("PR_AGENT_FORCE_REVIEW") == "true" or not any(label.get("name") == "skip-pr-agent" for label in pr.get("labels", [])))
    )

def install_commit_bound_publisher(number, revision):
    from pr_agent.git_providers.github_provider import GithubProvider
    from pr_agent.tools.pr_reviewer import PRReviewer
    receipts = []
    ready = [None]
    marker = f"<!-- pr-agent:kimi-k2.7-code:{revision}:actions-v1-ef96f0042792489d -->"
    prepare = PRReviewer._prepare_pr_review

    def validated(self, *args, **kwargs):
        text = prepare(self, *args, **kwargs)
        if not isinstance(text, str) or not text.strip():
            raise ValueError("No validated review to publish")
        # The pinned reviewer may add its native full-review identity after
        # rendering. Accept that exact transformation, never a stripped banner.
        ready[0] = {text.strip()}
        if self.git_provider.supports_review_comment_identity() is True:
            from pr_agent.algo.comment_identity import PRReviewIdentity, add_pr_review_identity
            ready[0].add(add_pr_review_identity(text, PRReviewIdentity.REGULAR.value, self.git_provider).strip())
        return text

    PRReviewer._prepare_pr_review = validated

    def publish(self, body, is_temporary=False):
        if is_temporary:
            return None
        if not ready[0] or receipts or not isinstance(body, str) or body.strip() not in ready[0]:
            raise ValueError("Refusing incomplete review publication")
        if len(body) > 60000:
            raise ValueError("Review exceeds publication budget")
        if self.last_commit_id.sha != revision or self.pr.number != int(number):
            raise StalePullRequest("Provider snapshot differs from triggering revision")
        for index, review in enumerate(self.pr.get_reviews()):
            if index >= 1000:
                raise RuntimeError("Review history exceeds duplicate-check budget")
            if (
                getattr(getattr(review, "user", None), "login", "") == "github-actions[bot]" and os.environ.get("PR_AGENT_FORCE_REVIEW") != "true"
                and review.commit_id == revision
                and marker in (review.body or "")
                and review.state == "COMMENTED"
                and (review.body or "").startswith("## PR-Agent source review coverage")
            ):
                receipts.append({"published": False, "review_id": review.id, "deduplicated": True})
                return review
        # Check after inference and history reads. commit= binds even a last-moment
        # head change to the reviewed snapshot rather than implying a current-head review.
        if not eligible(fetch_pr(number), revision):
            raise StalePullRequest("PR changed before publication")
        review = self.pr.create_review(
            commit=self.last_commit_id,
            event="COMMENT",
            body=f"{body}\n\nReviewed commit `{revision}` using `{MODEL}`.\n{marker}",
        )
        if review.commit_id != revision or review.state != "COMMENTED":
            raise RuntimeError("GitHub did not confirm a comment review on the expected commit")
        receipts.append({"published": True, "review_id": review.id, "deduplicated": False})
        return review

    GithubProvider.publish_comment = publish
    return receipts

# Trusted runtime limits and full-patch packing for the pinned PR-Agent image.
import asyncio
import signal
import time

REVIEW_STATE = {"calls_started": 0, "calls_completed": 0, "calls_failed": 0, "category": None, "last_failure": None, "truncation_retries": 0, "recovered_truncations": 0}


class ReviewRequestTimeout(RuntimeError):
    pass


class ReviewDeadlineExceeded(RuntimeError):
    pass


class ReviewCallBudgetExceeded(RuntimeError):
    pass


def coverage_exit_status(coverage):
    return 2 if coverage["status"] == "partial" else 0


def failure_details(error):
    names, status, seen = [], None, set()
    current = error
    while current is not None and id(current) not in seen and len(names) < 6:
        seen.add(id(current))
        names.append(type(current).__name__)
        code = getattr(current, "status_code", None)
        if status is None and isinstance(code, int) and 100 <= code <= 599:
            status = code
        current = current.__cause__ or current.__context__
    kinds = set(names)
    if "EmptyTruncatedResponseError" in kinds or "ReviewOutputTruncated" in kinds:
        category = "output_truncated"
    elif kinds & {"ReviewRequestTimeout", "APITimeoutError", "Timeout", "TimeoutError", "ReadTimeout", "ConnectTimeout"}:
        category = "request_timeout"
    elif status in (408, 504, 524):
        category = "provider_timeout"
    elif status == 429 or "RateLimitError" in kinds:
        category = "provider_rate_limit"
    elif status == 401 or "AuthenticationError" in kinds:
        category = "provider_authentication"
    elif status == 403 or "PermissionDeniedError" in kinds:
        category = "provider_permission"
    elif status is not None and status >= 500:
        category = "provider_unavailable"
    elif status is not None and status >= 400:
        category = "provider_request_rejected"
    else:
        category = next((value for name, value in (
            ("StalePullRequest", "stale_pr"),
            ("ReviewDeadlineExceeded", "review_deadline"),
            ("ReviewCallBudgetExceeded", "call_budget"),
            ("ReviewIncompleteResponse", "incomplete_completion"),
            ("APIConnectionError", "provider_connection"),
            ("APIError", "provider_error"),
            ("ValueError", "invalid_review_or_input"),
        ) if name in kinds), "review_failed")
    return {"category": category, "error_types": names, "http_status": status}


def safe_failure_category(error, *, use_recorded=True):
    if use_recorded and REVIEW_STATE["category"]:
        return REVIEW_STATE["category"]
    return failure_details(error)["category"]


class ReviewOutputTruncated(RuntimeError):
    pass


class ReviewIncompleteResponse(RuntimeError):
    pass


def arm_review_deadline(seconds=540):
    def expired(*_):
        REVIEW_STATE["category"] = "review_deadline"
        raise ReviewDeadlineExceeded("Review wall deadline reached")
    signal.signal(signal.SIGALRM, expired)
    signal.alarm(seconds)


def disable_handler_retries():
    from pr_agent.algo.ai_handlers.litellm_ai_handler import LiteLLMAIHandler
    original = LiteLLMAIHandler._chat_completion_with_retry
    # config.num_retries controls the client, not this separate Tenacity loop.
    # Copy its retry policy with an unconditional stop after the first attempt.
    LiteLLMAIHandler._chat_completion_with_retry = original.retry_with(stop=lambda _state: True)
    return original


def require_complete_responses(*, guard=None, timeout_seconds=420, max_calls=4, max_parallel=4,
                               deadline_seconds=540, minimum_retry_seconds=60):
    from pr_agent.algo.ai_handlers.litellm_ai_handler import LiteLLMAIHandler
    original = LiteLLMAIHandler.chat_completion
    semaphore = asyncio.Semaphore(max_parallel)
    deadline = time.monotonic() + deadline_seconds

    def failed(index, started, error):
        REVIEW_STATE["calls_failed"] += 1
        REVIEW_STATE["last_failure"] = failure_details(error)
        print(json.dumps({"event": "pr_agent_call_failed", "call": index,
                          **failure_details(error),
                          "elapsed_seconds": round(time.monotonic() - started, 1)}), flush=True)

    async def complete(self, *args, **kwargs):
        async with semaphore:
            previous_error = None
            for attempt in range(2):
                if REVIEW_STATE["category"] in ("review_deadline", "call_budget"):
                    raise ReviewCallBudgetExceeded("Review limit reached")
                if REVIEW_STATE["calls_started"] >= max_calls:
                    if previous_error is not None:
                        raise previous_error
                    REVIEW_STATE["category"] = "call_budget"
                    raise ReviewCallBudgetExceeded("Review call limit reached")
                if guard is not None and not await asyncio.to_thread(guard):
                    REVIEW_STATE["category"] = "stale_pr"
                    raise StalePullRequest("PR changed while waiting for a review slot")
                # The guard yields to other chunks. Reserve both budgets only after
                # it returns, without another await between checking and reserving.
                remaining = deadline - time.monotonic()
                if attempt and (REVIEW_STATE["truncation_retries"] >= 1
                                or remaining < minimum_retry_seconds
                                or REVIEW_STATE["calls_started"] >= max_calls):
                    raise previous_error
                if REVIEW_STATE["calls_started"] >= max_calls:
                    REVIEW_STATE["category"] = "call_budget"
                    raise ReviewCallBudgetExceeded("Review call limit reached")
                if remaining <= 0:
                    REVIEW_STATE["category"] = "review_deadline"
                    raise ReviewDeadlineExceeded("Review wall deadline reached")
                if attempt:
                    REVIEW_STATE["truncation_retries"] += 1
                    print(json.dumps({"event": "pr_agent_truncation_recovery_started",
                                      "remaining_calls": max_calls - REVIEW_STATE["calls_started"]}), flush=True)
                REVIEW_STATE["calls_started"] += 1
                index = REVIEW_STATE["calls_started"]
                started = time.monotonic()
                print(json.dumps({"event": "pr_agent_call_started", "call": index}), flush=True)
                try:
                    text, finish = await asyncio.wait_for(original(self, *args, **kwargs),
                                                          min(timeout_seconds, remaining))
                    if not isinstance(text, str) or not text.strip() or finish != "stop":
                        if finish == "length":
                            raise ReviewOutputTruncated("Model output reached its token limit")
                        raise ReviewIncompleteResponse("Incomplete model response")
                except asyncio.TimeoutError as error:
                    REVIEW_STATE["category"] = "request_timeout"
                    failed(index, started, error)
                    raise ReviewRequestTimeout("Review request deadline reached") from error
                except Exception as error:
                    REVIEW_STATE["category"] = safe_failure_category(error, use_recorded=False)
                    failed(index, started, error)
                    if (not attempt and REVIEW_STATE["category"] == "output_truncated"
                            and REVIEW_STATE["truncation_retries"] < 1
                            and REVIEW_STATE["calls_started"] < max_calls
                            and deadline - time.monotonic() >= minimum_retry_seconds):
                        previous_error = error
                        continue
                    raise
                else:
                    REVIEW_STATE["calls_completed"] += 1
                    if attempt:
                        REVIEW_STATE["recovered_truncations"] += 1
                        # Do not let a recovered call mask a later publication error,
                        # or clear an unrecovered failure from another chunk.
                        if REVIEW_STATE["calls_failed"] == REVIEW_STATE["recovered_truncations"]:
                            if REVIEW_STATE["category"] == "output_truncated":
                                REVIEW_STATE["category"] = None
                            REVIEW_STATE["last_failure"] = None
                    print(json.dumps({"event": "pr_agent_call_completed", "call": index,
                                      "elapsed_seconds": round(time.monotonic() - started, 1)}), flush=True)
                    return text, finish

    LiteLLMAIHandler.chat_completion = complete
    return original


def install_full_patch_packing():
    import pr_agent.algo.pr_processing as processing
    original = processing._pack_pr_multi_diffs
    original_deletions = processing.handle_patch_deletions
    original_render = processing.decouple_and_convert_to_hunks_with_lines_numbers
    # Keep deletion-only hunks and deleted-file bodies in the reviewed input.
    processing.handle_patch_deletions = lambda patch, *_args: patch
    def render(patch, file):
        # Native rendering replaces deleted bodies with a filename-only notice.
        # Preserve old hunks too; only filename is otherwise consumed by this seam.
        from types import SimpleNamespace
        return original_render(patch, SimpleNamespace(filename=file.filename) if file else file)
    processing.decouple_and_convert_to_hunks_with_lines_numbers = render

    def pack(file_dict, token_handler, max_calls, return_remaining_files, token_budget, *, return_coverage=False):
        # Count exactly as the native packer, including stripped text, to preserve
        # the model-bound prompt/output reserve calculated by PR-Agent.
        def count(text):
            return max(token_handler.count_tokens(text), token_handler.count_tokens(text.strip()))

        def split_patch(filename, patch):
            if count(patch) <= token_budget:
                return [patch]
            first_hunk = re.search(r"(?m)^@@ ", patch)
            if not first_hunk:
                return []  # Unsupported oversized representation: disclose, never clip.
            header, body = patch[:first_hunk.start()], patch[first_hunk.start():]
            if count(header) >= token_budget:
                return []
            lines = body.splitlines(keepends=True)
            states, hunk, side = [], "", ""
            for line in lines:
                if line.startswith("@@ "):
                    hunk, side = line, ""
                    states.append("")
                elif line.strip() in ("__new hunk__", "__old hunk__"):
                    side = line
                    states.append(hunk)
                else:
                    states.append(hunk + side)
            parts, start = [], 0
            while start < len(lines):
                prefix = header + states[start]
                low, high, fit = start + 1, len(lines), start
                while low <= high:
                    end = (low + high) // 2
                    candidate = prefix + "".join(lines[start:end])
                    if count(candidate) <= token_budget:
                        fit, low = end, end + 1
                    else:
                        high = end - 1
                if fit == start:
                    return []  # One indivisible source line cannot fit; disclose the file.
                parts.append(prefix + "".join(lines[start:fit]))
                start = fit
            return parts

        def priority(item):
            path = item[0]
            if path.startswith(("docs/", "changelog/")) or path.endswith(".md"):
                return 2
            if "test" in path.rsplit("/", 1)[-1] or "/tests/" in path:
                return 1
            return 0

        def build(budget):
            nonlocal token_budget
            token_budget = budget
            chunks, current, totals, included = [], "", {}, {}
            for filename, data in sorted(file_dict.items(), key=priority):
                parts = [] if data.get("content_fetch_failed") or not data["patch"] else split_patch(filename, data["patch"])
                totals[filename] = len(parts)
                included[filename] = 0
                for part in parts:
                    if current and count(current + "\n" + part) > token_budget:
                        chunks.append(current)
                        current = ""
                    if len(chunks) >= max_calls:
                        break
                    current = current + "\n" + part if current else part
                    included[filename] += 1
            if current and len(chunks) < max_calls:
                chunks.append(current)
            remaining = [name for name in file_dict if not included[name]]
            partial = [name for name in file_dict if included[name] and included[name] < totals[name]]
            return chunks, remaining, partial

        native_budget = token_budget
        chunks, remaining, partial = build(native_budget)
        # A model's maximum context is not a good target for each review call.
        # Prefer smaller complete prompts, but never add packing omissions relative
        # to the native-budget result just to reach that target.
        if native_budget > 24000 and not remaining and not partial:
            budget = 24000
            while True:
                candidate = build(budget)
                preferred_calls = max_calls - 1 if max_calls >= 4 else max_calls
                if not candidate[1] and not candidate[2] and len(candidate[0]) <= preferred_calls:
                    chunks, remaining, partial = candidate
                    break
                if budget == native_budget:
                    break
                budget = min(native_budget, budget * 3 // 2)
        if return_coverage:
            return processing.PackedPRDiffs(chunks, remaining, partial)
        return (chunks, remaining) if return_remaining_files else chunks

    # Native review only enters chunking when its maximum-context diff omits
    # files. Large prompts that still fit that maximum otherwise bypass packing.
    import importlib
    reviewer_module = importlib.import_module("pr_agent.tools.pr_reviewer")
    original_diff = getattr(reviewer_module, "get_pr_diff", None)
    if original_diff is not None:
        def prepare_diff(provider, token_handler, model, **kwargs):
            output = original_diff(provider, token_handler, model, **kwargs)
            if (isinstance(output, processing.PreparedPRDiff)
                    and not output.remaining_files_list
                    and token_handler.count_tokens(output.diff) > 24000):
                from dataclasses import replace
                from pr_agent.config_loader import get_settings
                prepared = output
                if output.file_dict is None:
                    # The native fast path deliberately does not prepare file data.
                    # Build it through the same native seam used for large diffs,
                    # preserving its real model-bound capacities and token counts.
                    files = provider.get_diff_files()
                    languages = processing.sort_files_by_main_languages(provider.get_languages(), files)
                    budget = output.attempt_budget
                    soft = budget.available_tokens(processing.OUTPUT_BUFFER_TOKENS_SOFT_THRESHOLD,
                                                   preserve_minimum=True, clamp=False)
                    hard = budget.available_tokens(processing.OUTPUT_BUFFER_TOKENS_HARD_THRESHOLD,
                                                   preserve_minimum=True, clamp=False)
                    _, _, _, _, file_dict, _ = processing.pr_generate_compressed_diff(
                        languages, budget.token_handler, soft, hard, True, True)
                    prepared = replace(output, file_dict=file_dict,
                                       files_by_name={file.filename: file for file in files})
                options = {"max_calls": get_settings().pr_reviewer.max_number_of_calls,
                           "add_line_numbers": True, "return_coverage": True,
                           "include_filtered_file_names": False, "prepared_diff": prepared}
                if "output_token_reserve" in kwargs:
                    options["output_token_reserve"] = kwargs["output_token_reserve"]
                plan = processing.get_pr_multi_diffs(provider, token_handler, model, **options)
                if len(plan.chunks) > 1:
                    # These real selected files are pending until native chunk
                    # review replaces this list with the actual packing coverage.
                    return replace(prepared, remaining_files_list=list(prepared.file_dict))
            return output
        reviewer_module.get_pr_diff = prepare_diff
    processing._pack_pr_multi_diffs = pack
    return original, original_deletions, original_render


def main():
    if REPOSITORY not in json.loads(os.environ["PR_ALLOWED_REPOSITORIES"]):
        raise ValueError("Repository is outside the review allowlist")
    number = os.environ["PR_NUMBER"]
    revision = os.environ["PR_REVISION"]
    if not re.fullmatch(r"[1-9][0-9]*", number):
        raise ValueError("Invalid PR number")
    if not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise ValueError("Invalid revision")
    token = os.environ["GITHUB__USER_TOKEN"]
    if not token or not os.environ.get("OPENROUTER__KEY"):
        raise ValueError("Missing review credential")
    publish = os.environ.get("PR_AGENT_PUBLISH_REVIEWS", "false") == "true"
    if not eligible(fetch_pr(number), revision):
        print("Skipped: PR is stale or no longer eligible for the pilot.")
        return 0

    from pr_agent import cli
    from pr_agent.config_loader import get_settings

    settings = get_settings()
    # Host-owned settings: PR contents cannot redirect inference or choose publication.
    for key, value in {
        "config.git_provider": "github",
        "config.model": MODEL,
        "config.model_weak": MODEL,
        "config.model_reasoning": MODEL,
        "config.reasoning_effort": "low",
        "config.fallback_models": [],
        "config.custom_model_max_tokens": 131072,
        "config.max_model_tokens": 131072,
        "config.ai_timeout": 420,
        "config.num_retries": 0,
        "config.retry_same_model_on_timeout": False,
        "config.propagate_tool_errors": True,
        "config.publish_output": publish,
        "config.publish_output_progress": False,
        "config.enable_auto_approval": False,
        "config.use_repo_settings_file": False,
        "config.use_global_settings_file": False,
        "config.enable_per_directory_settings": False,
        "config.verbosity_level": 0,
        "config.repo_context_sibling_repos": [],
        "openrouter.api_base": "https://openrouter.ai/api/v1",
        "openrouter.max_tokens": 32768,
        # Explicit provider control avoids relying on bundled model metadata.
        "openrouter.reasoning_effort": "low",
        "model_routing.enable": False,
        "pr_reviewer.enable_review_labels_security": False,
        "pr_reviewer.enable_review_labels_effort": False,
        "pr_reviewer.publish_review_failure_comment": False,
        "pr_reviewer.enable_large_pr_chunking": True,
        "pr_reviewer.max_number_of_calls": 4,
        "ignore.glob": review_policy().get("ignore_globs", []),
        "ignore.regex": [],
        "config.ignore_language_framework": [],
        "pr_reviewer.require_ticket_analysis_review": False,
        "pr_reviewer.persistent_comment": False,
        "pr_reviewer.persistent_finding_state": False,
        "pr_reviewer.inline_key_issues": False,
        "pr_reviewer.final_update_message": False,
    }.items():
        settings.set(key, value)
    install_contract_expansion()
    _, coverage_reports = install_coverage_reporting()
    install_full_patch_packing()
    disable_handler_retries()
    require_complete_responses(guard=lambda: eligible(fetch_pr(number), revision))
    arm_review_deadline()
    receipts = install_commit_bound_publisher(number, revision) if publish else []
    with tempfile.TemporaryDirectory(prefix="pr-agent-") as directory:
        output = Path(directory) / "review.json"
        result = cli.run(inargs=[
            "--pr_url", f"https://github.com/{REPOSITORY}/pull/{number}",
            "--json-output", str(output), "review",
        ])
        data = json.loads(output.read_text())
        if not isinstance(data.get("review"), dict) or not data["review"]:
            raise ValueError("PR-Agent did not produce a structured review")
        if len(coverage_reports) != 1:
            raise ValueError("Expected one complete coverage inventory")
        # Upstream JSON export retains only its two native coverage lists.
        # Use the same host inventory that prefixed the published review.
        data["coverage"] = coverage_reports[0]
        # The native CLI returns failure even after publishing valid
        # successful chunks. Retain that explicit partial outcome,
        # while all other failures remain fail-closed.
        if result not in (None, 0) and not (
            result == 1
            and data["coverage"]["status"] == "partial"
            and data["coverage"]["failed_chunks"] > 0
        ):
            raise RuntimeError("PR-Agent reported a failed review")
        if publish and len(receipts) != 1:
            raise RuntimeError("Expected one confirmed review publication or duplicate receipt")
        # Keep the diff and review text out of TaskRun logs and results.
        print(json.dumps({
            "event": "pr_agent_review_completed",
            "model": MODEL,
            "revision": revision,
            "published": any(receipt["published"] for receipt in receipts),
            "publication": receipts,
            "usage": data.get("usage", {}),
            "coverage": data.get("coverage", {}),
            "execution": REVIEW_STATE,
            "outcome": "coverage_incomplete" if coverage_exit_status(data["coverage"]) else "complete",
        }))
    return coverage_exit_status(data["coverage"])

if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as error:
        # Provider errors may contain private content; report only the type.
        print(json.dumps({"event": "pr_agent_failed",
                          **(REVIEW_STATE.get("last_failure") or failure_details(error)),
                          "category": safe_failure_category(error),
                          "calls_started": REVIEW_STATE["calls_started"],
                          "calls_completed": REVIEW_STATE["calls_completed"],
                          "calls_failed": REVIEW_STATE["calls_failed"]}), file=sys.stderr)
        sys.exit(1)
