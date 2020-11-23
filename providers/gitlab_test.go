package providers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/apis/sessions"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func testGitLabProvider(hostname string) *GitLabProvider {
	p := NewGitLabProvider(
		&ProviderData{
			ProviderName: "",
			LoginURL:     &url.URL{},
			RedeemURL:    &url.URL{},
			ProfileURL:   &url.URL{},
			ValidateURL:  &url.URL{},
			Scope:        ""})
	if hostname != "" {
		updateURL(p.Data().LoginURL, hostname)
		updateURL(p.Data().RedeemURL, hostname)
		updateURL(p.Data().ProfileURL, hostname)
		updateURL(p.Data().ValidateURL, hostname)
	}

	return p
}

func testGitLabBackend() *httptest.Server {
	userInfo := `
		{
			"nickname": "FooBar",
			"email": "foo@bar.com",
			"email_verified": false,
			"groups": ["foo", "bar"]
		}
	`
	projectInfo := `
		{
			"name": "MyProject",
			"archived": false,
			"path_with_namespace": "my_group/my_project"
		}
	`
	authHeader := "Bearer gitlab_access_token"

	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/oauth/userinfo":
				if r.Header["Authorization"][0] == authHeader {
					w.WriteHeader(200)
					w.Write([]byte(userInfo))
				} else {
					w.WriteHeader(401)
				}
			case "/api/v4/projects/my_group/my_project":
				if r.Header["Authorization"][0] == authHeader {
					w.WriteHeader(200)
					w.Write([]byte(projectInfo))
				} else {
					w.WriteHeader(401)
				}
			case "/api/v4/projects/my_group/my_bad_project":
				w.WriteHeader(403)
			default:
				w.WriteHeader(404)
			}
		}))
}

var _ = Describe("Gitlab Provider Tests", func() {
	var p *GitLabProvider
	var b *httptest.Server

	BeforeEach(func() {
		b = testGitLabBackend()

		bURL, err := url.Parse(b.URL)
		Expect(err).To(BeNil())

		p = testGitLabProvider(bURL.Host)
	})

	AfterEach(func() {
		b.Close()
	})

	Context("with bad token", func() {
		It("should trigger an error", func() {
			p.AllowUnverifiedEmail = false
			session := &sessions.SessionState{AccessToken: "unexpected_gitlab_access_token"}
			err := p.EnrichSession(context.Background(), session)
			Expect(err).To(MatchError(errors.New("failed to retrieve user info: error getting user info: unexpected status \"401\": ")))
		})
	})

	Context("when filtering on email", func() {
		type emailsTableInput struct {
			expectedError        error
			expectedValue        string
			domains              []string
			allowUnverifiedEmail bool
		}

		DescribeTable("should return expected results",
			func(in emailsTableInput) {
				p.AllowUnverifiedEmail = in.allowUnverifiedEmail
				session := &sessions.SessionState{AccessToken: "gitlab_access_token"}

				if in.domains != nil {
					if len(in.domains) >= 1 {
						p.EmailDomains = in.domains
					}
				}

				err := p.EnrichSession(context.Background(), session)

				if in.expectedError != nil {
					Expect(err).To(MatchError(err))
				} else {
					Expect(err).To(BeNil())
					Expect(session.Email).To(Equal(in.expectedValue))
				}
			},
			Entry("unverified email denied", emailsTableInput{
				expectedError:        errors.New("user email is not verified"),
				allowUnverifiedEmail: false,
			}),
			Entry("unverified email allowed", emailsTableInput{
				expectedError:        nil,
				expectedValue:        "foo@bar.com",
				allowUnverifiedEmail: true,
			}),
		)
	})

	Context("when filtering on gitlab entities (groups and projects)", func() {
		type entitiesTableInput struct {
			expectedError error
			expectedValue []string
			projects      []string
			groups        []string
		}

		DescribeTable("should return expected results",
			func(in entitiesTableInput) {
				p.AllowUnverifiedEmail = true
				session := &sessions.SessionState{AccessToken: "gitlab_access_token"}

				p.Projects = in.projects
				p.SetProjectScope()

				if len(in.groups) > 0 {
					p.Groups = in.groups
				}

				err := p.EnrichSession(context.Background(), session)

				if in.expectedError != nil {
					Expect(err).To(MatchError(in.expectedError))
				} else {
					Expect(err).To(BeNil())
					Expect(session.Groups).To(Equal(in.expectedValue))
				}
			},
			Entry("project membership valid", entitiesTableInput{
				expectedError: nil,
				expectedValue: []string{"my_group/my_project"},
				projects:      []string{"my_group/my_project"},
			}),
			Entry("project membership invalid", entitiesTableInput{
				expectedError: nil,
				expectedValue: nil,
				projects:      []string{"my_group/my_bad_project"},
			}),
			Entry("group membership valid", entitiesTableInput{
				expectedError: nil,
				expectedValue: nil,
				groups:        []string{"foo"},
			}),
		)

	})

})
