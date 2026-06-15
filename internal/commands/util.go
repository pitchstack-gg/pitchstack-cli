package commands

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
)

func boolPtr(cmd *cli.Command, name string) *bool {
	if !cmd.IsSet(name) {
		return nil
	}
	v := cmd.Bool(name)
	return &v
}

func int32Ptr(cmd *cli.Command, name string) *int32 {
	if !cmd.IsSet(name) {
		return nil
	}
	v := int32(cmd.Int(name))
	return &v
}

func float64Ptr(cmd *cli.Command, name string) *float64 {
	if !cmd.IsSet(name) {
		return nil
	}
	v := cmd.Float64(name)
	return &v
}

func parseResourceDescriptor(spec string) (resourceType string, id string, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", "", errors.New("empty")
	}
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New("expected <type>:<id>")
	}
	resourceType = strings.TrimSpace(parts[0])
	id = strings.TrimSpace(parts[1])
	if resourceType == "" || id == "" {
		return "", "", errors.New("expected <type>:<id>")
	}
	return resourceType, id, nil
}

func parseInt32(s string) (*int32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	vs := int32(v)
	return &vs, nil
}

func parseStringMap(values []string) (map[string]string, error) {
	out := map[string]string{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			key, val, ok := strings.Cut(part, "=")
			if !ok {
				return nil, errors.New("expected key=value")
			}
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			if key == "" {
				return nil, errors.New("metadata key must not be empty")
			}
			out[key] = val
		}
	}
	return out, nil
}

func setQueryString(query url.Values, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		query.Set(key, value)
	}
}

func pathWithQuery(path string, query url.Values) string {
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}
