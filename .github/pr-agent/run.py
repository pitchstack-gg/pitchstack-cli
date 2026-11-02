"""Trusted default-branch Actions coordinator; PR contents are never executed."""
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parent
CHECK_NAME = "PR-Agent review"


def api(repository, path, data=None, method=None):
    request = Request(f"https://api.github.com/repos/{repository}/{path}",
        data=json.dumps(data).encode() if data is not None else None,
        method=method, headers={"Authorization": f"Bearer {os.environ['GITHUB_TOKEN']}",
        "Accept": "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28"})
    with urlopen(request, timeout=30) as response:
        return json.load(response)


def eligible(pr, repository, revision, force=False):
    return (pr.get("state") == "open" and not pr.get("draft", True)
        and pr.get("base", {}).get("ref") == "main"
        and pr.get("base", {}).get("repo", {}).get("full_name") == repository
        and (pr.get("head", {}).get("repo") or {}).get("full_name") == repository
        and pr.get("head", {}).get("sha") == revision
        and (force or not any(x.get("name") == "skip-pr-agent" for x in pr.get("labels", []))))


def state_path():
    return Path(os.environ["RUNNER_TEMP"]) / f"pr-agent-{os.environ['GITHUB_RUN_ID']}-{os.environ.get('GITHUB_RUN_ATTEMPT', '1')}.json"


def output(key, value):
    with open(os.environ["GITHUB_OUTPUT"], "a") as file:
        file.write(f"{key}={value}\n")


def finish(state, conclusion, title, summary):
    api(state["repository"], f"check-runs/{state['check_id']}", {
        "status": "completed", "conclusion": conclusion,
        "output": {"title": title, "summary": summary}}, method="PATCH")


def prepare(config):
    repository = os.environ["GITHUB_REPOSITORY"]
    if repository != config["repository"]:
        raise ValueError("Repository differs from trusted runner configuration")
    event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text())
    number = str((event.get("pull_request") or {}).get("number") or os.environ.get("PR_INPUT_NUMBER", ""))
    if not re.fullmatch(r"[1-9][0-9]*", number):
        raise ValueError("Invalid PR number")
    pr = api(repository, f"pulls/{number}")
    revision = (event.get("pull_request") or {}).get("head", {}).get("sha") or pr["head"]["sha"]
    force = os.environ.get("PR_AGENT_FORCE_REVIEW") == "true"
    if not re.fullmatch(r"[0-9a-f]{40}", revision) or not eligible(pr, repository, revision, force):
        output("eligible", "false")
        return
    check = api(repository, "check-runs", {"name": CHECK_NAME, "head_sha": revision,
        "status": "in_progress", "external_id": f"pr-agent:{number}:{revision}:{os.environ['GITHUB_RUN_ID']}",
        "details_url": f"https://github.com/{repository}/actions/runs/{os.environ['GITHUB_RUN_ID']}"})
    state = {"repository": repository, "number": number, "revision": revision, "check_id": check["id"]}
    state_path().write_text(json.dumps(state))
    marker = f"<!-- pr-agent:kimi-k2.7-code:{revision}:{config['policy']} -->"
    if not force:
        for page in range(1, 11):
            reviews = api(repository, f"pulls/{number}/reviews?per_page=100&page={page}")
            for review in reviews:
                if (review.get("user", {}).get("login") == "github-actions[bot]"
                        and review.get("commit_id") == revision and review.get("state") == "COMMENTED"
                        and marker in (review.get("body") or "")
                        and (review.get("body") or "").startswith("## PR-Agent source review coverage")):
                    finish(state, "success", "Reused complete review", f"A complete review already covers `{revision}` under `{config['policy']}`.")
                    output("eligible", "false")
                    return
            if len(reviews) < 100:
                break
        else:
            raise ValueError("Review history exceeds duplicate-check budget")
    output("eligible", "true")


def run(config):
    state = json.loads(state_path().read_text())
    if not os.environ.get("OPENROUTER_API_KEY"):
        raise ValueError("PR-Agent OpenRouter credential is unavailable")
    env = {**os.environ, "OPENROUTER__KEY": os.environ["OPENROUTER_API_KEY"],
        "GITHUB__USER_TOKEN": os.environ["GITHUB_TOKEN"], "PR_NUMBER": state["number"],
        "PR_REVISION": state["revision"], "PR_REPOSITORY": state["repository"],
        "PR_ALLOWED_REPOSITORIES": json.dumps(config["allowed_repositories"]),
        "PR_REVIEW_POLICIES": json.dumps(config["review_policies"]), "PR_AGENT_PUBLISH_REVIEWS": "true",
        "LITELLM_LOCAL_MODEL_COST_MAP": "True", "LOG_LEVEL": "CRITICAL", "PYTHONPATH": "/app"}
    names = ["OPENROUTER__KEY", "GITHUB__USER_TOKEN", "PR_NUMBER", "PR_REVISION", "PR_REPOSITORY",
        "PR_ALLOWED_REPOSITORIES", "PR_REVIEW_POLICIES", "PR_AGENT_PUBLISH_REVIEWS",
        "PR_CONTRACT_READ_TOKEN", "PR_AGENT_FORCE_REVIEW", "LITELLM_LOCAL_MODEL_COST_MAP", "LOG_LEVEL", "PYTHONPATH"]
    container = f"pr-agent-{os.environ['GITHUB_RUN_ID']}-{os.environ.get('GITHUB_RUN_ATTEMPT', '1')}"
    command = ["docker", "run", "--rm", "--name", container, "--cpus", "2", "--memory", "2g", "--entrypoint", "python"]
    for name in names:
        if name in env:
            command += ["-e", name]
    command += ["-v", f"{ROOT / 'review.py'}:/review.py:ro", config["image"], "/review.py"]
    # Capture provider/native output privately; emit only the trusted structured receipt.
    with tempfile.TemporaryFile() as stdout, tempfile.TemporaryFile() as stderr:
        try:
            result = subprocess.run(command, env=env, stdout=stdout, stderr=stderr, timeout=660)
        finally:
            subprocess.run(["docker", "rm", "--force", container], capture_output=True)
        stdout.seek(0)
        receipt = None
        failure = None
        for raw in stdout:
            try:
                item = json.loads(raw)
            except (ValueError, UnicodeError):
                continue
            if isinstance(item, dict) and item.get("event") == "pr_agent_review_completed":
                receipt = item
        stderr.seek(0)
        for raw in stderr:
            try:
                item = json.loads(raw)
            except (ValueError, UnicodeError):
                continue
            if isinstance(item, dict) and item.get("event") == "pr_agent_failed":
                allowed = {"output_truncated", "request_timeout", "provider_timeout", "provider_rate_limit",
                    "provider_authentication", "provider_permission", "provider_unavailable", "provider_request_rejected",
                    "review_failed", "provider_connection", "provider_error", "invalid_review_or_input", "stale_pr",
                    "review_deadline", "call_budget", "incomplete_completion"}
                if item.get("category") in allowed:
                    failure = item["category"]
        if failure:
            print(json.dumps({"event": "pr_agent_actions_provider_failed", "category": failure}), flush=True)
    if (result.returncode not in (0, 2) or receipt is None
            or receipt.get("revision") != state["revision"]
            or len(receipt.get("publication", [])) != 1):
        finish(state, "failure", "Review did not complete", f"PR-Agent failed or the PR became ineligible. No complete review is claimed. Category: `{failure or 'review_failed'}`.")
        raise RuntimeError("PR-Agent did not return a validated review receipt")
    coverage = receipt["coverage"]
    partial = coverage["status"] == "partial"
    failed = bool(coverage["failed_chunks"] or coverage["files_not_reviewed"]
                  or coverage["files_partially_reviewed"] or coverage.get("contract_expansion_failures"))
    safe = {"event": "pr_agent_actions_completed", "revision": state["revision"],
        "coverage": coverage, "execution": receipt.get("execution", {}), "published": receipt.get("published", False)}
    print(json.dumps(safe), flush=True)
    summary = (f"Reviewed commit `{state['revision']}`. Changed files: {coverage['changed_files']}; "
        f"reviewed patches: {len(coverage['files_reviewed'])}; failed chunks: {coverage['failed_chunks']}. "
        + ("Partial advisory review: consult the published coverage inventory. This is not a complete PR review." if partial else
           "All selected patches were reviewed. Generated exclusions remain subject to the repository's separate CI gates."))
    finish(state, "failure" if failed else "neutral" if partial else "success",
           "Incomplete review" if failed else "Partial advisory review" if partial else "Selected-patch review completed", summary)
    if failed:
        raise RuntimeError("Review has failed chunks or missing selected input")


def finalize():
    container = f"pr-agent-{os.environ['GITHUB_RUN_ID']}-{os.environ.get('GITHUB_RUN_ATTEMPT', '1')}"
    subprocess.run(["docker", "rm", "--force", container], capture_output=True)
    if not state_path().exists():
        return
    state = json.loads(state_path().read_text())
    check = api(state["repository"], f"check-runs/{state['check_id']}")
    if check.get("status") != "completed":
        cancelled = os.environ.get("PR_AGENT_JOB_STATUS") == "cancelled"
        finish(state, "cancelled" if cancelled else "failure", "Review interrupted", "The workflow ended before a validated review result was recorded.")


def main():
    config = json.loads((ROOT / "config.json").read_text())
    command = sys.argv[1]
    if command == "prepare":
        prepare(config)
    elif command == "run":
        run(config)
    elif command == "finalize":
        finalize()
    else:
        raise ValueError("Unknown command")


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        print(json.dumps({"event": "pr_agent_actions_failed", "error_type": type(error).__name__}), file=sys.stderr)
        sys.exit(1)
