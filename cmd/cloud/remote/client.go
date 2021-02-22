package remote

import (
	"fmt"
	"github.com/ory/x/httpx"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ory/kratos-client-go/client"
	"github.com/ory/x/cmdx"
	"github.com/ory/x/flagx"
	"github.com/ory/x/stringsx"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	FlagProject     = "project"
	FlagUpstream    = "upstream"
	projectEnvKey   = "ORY_PROJECT_ID"
	projectApiToken = "ORY_API_TOKEN"
)

type tokenTransporter struct {
	http.RoundTripper
	token string
}

func (t *tokenTransporter) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "bearer " + t.token)
	return t.RoundTripper.RoundTrip(req)
}

func NewHTTPClient(cmd *cobra.Command) *http.Client {
	token := os.Getenv(projectApiToken)
	if len(token) == 0 {
		cmdx.Fatalf(`Ory API Token could not be detected! Did you forget to set the environment variable "%s"?

You can create an API Token in the Ory Console. Once created, set the environment variable as follows.

**Unix (Linux, macOS)**

$ export ORY_API_TOKEN="<your-api-token-here>"
$ ory ...

**Windows (Powershell)**

> $env:ORY_API_TOKEN = '<your-api-token-here>'
> ory ...

**Windows (cmd.exe)**

> set ORY_API_TOKEN = "<your-api-token-here>"
> ory ...
`, projectApiToken)
		return nil
	}

	return &http.Client{
		Transport: httpx.NewResilientRoundTripper(&tokenTransporter{
			RoundTripper: http.DefaultTransport,
			token: token,
		}, time.Millisecond * 500, time.Second*30),
		Timeout:   time.Second * 10,
	}
}

func NewAdminClient(cmd *cobra.Command) *client.OryKratos {
	project := stringsx.Coalesce(flagx.MustGetString(cmd, FlagProject), os.Getenv(projectEnvKey))
	if project == "" {
		cmdx.Fatalf("You have to set the Ory Cloud Project ID, try --help for details.")
	}

	upstream, err := url.ParseRequestURI(flagx.MustGetString(cmd, FlagUpstream))
	if err != nil {
		cmdx.Must(err, "Unable to parse upstream URL because: %s", err)
	}

	return client.NewHTTPClientWithConfig(nil, &client.TransportConfig{
		Host:     fmt.Sprintf("%s.projects.%s", project, upstream.Host),
		BasePath: "/api/kratos/admin/",
		Schemes:  []string{upstream.Scheme},
	})
}

func RegisterClientFlags(flags *pflag.FlagSet) {
	flags.StringP(FlagProject, FlagProject[:1], "", fmt.Sprintf("Set your Ory Cloud Project ID. Alternatively set using the %s environmental variable.", projectEnvKey))
	flags.String(FlagUpstream, "https://oryapis.com", "Use a different upstream domain.")
}
