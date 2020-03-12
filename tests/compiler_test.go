package tests

import (
	"github.com/isbm/nano-cms/nanostate/compiler"
	"gopkg.in/check.v1"
	"testing"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type CompilerTestSuite struct {
	cmp *nanocms_compiler.NstCompiler
}

var _ = check.Suite(&CompilerTestSuite{})

func (s *CompilerTestSuite) SetUpSuite(c *check.C)    {}
func (s *CompilerTestSuite) TearDownSuite(c *check.C) {}

func (s *CompilerTestSuite) SetUpTest(c *check.C) {
	s.cmp = nanocms_compiler.NewNstCompiler()
	err := s.cmp.LoadFile("states/definition.st")
	if err != nil {
		panic(err)
	}

	for {
		id := s.cmp.Cycle()
		if id == "" {
			break
		}
		err := s.cmp.LoadFile("states/pgsql.st")
		if err != nil {
			panic(err)
		}
	}
}

func (s *CompilerTestSuite) TearDownTest(c *check.C) {
	s.cmp = nil
}

/*
Test "add-some-more-users" has three expected entries.
*/
func (s *CompilerTestSuite) TestDefinitionAddSomeMoreUsersLen(c *check.C) {
	users := s.cmp.Tree().GetBranch("state").GetList("add-some-more-users")
	c.Assert(len(users), check.Equals, 3)
}

/*
Test "add-some-more-users" has proper ordering.
*/
func (s *CompilerTestSuite) TestDefinitionAddSomeMoreUsersOrdering(c *check.C) {
	users := s.cmp.Tree().GetBranch("state").GetList("add-some-more-users")
	names := []string{"john", "fred", "ralf"}
	for idx, kwset := range users {
		for _, k := range kwset.(*nanocms_compiler.OTree).Keys() {
			c.Assert(k, check.Equals, "system.user")
			c.Assert(kwset.(*nanocms_compiler.OTree).Get(k, nil).(*nanocms_compiler.OTree).Get("name", nil), check.Equals, names[idx])
		}
	}
	//c.Log(s.cmp.Tree().ToYAML())
}

/*
Test "installing Emacs on Debian using apt" .
*/
func (s *CompilerTestSuite) TestDefinitionInstallEmacsDebian(c *check.C) {
	c.Assert(len(s.cmp.Tree().GetBranch("state").GetList("install-emacs-apt")), check.Equals, 1)
}

/*
Test "installing Emacs on Debian not using yum" .
*/
func (s *CompilerTestSuite) TestDefinitionInstallEmacsRedhat(c *check.C) {
	c.Assert(s.cmp.Tree().GetBranch("state").Get("install-emacs-yum", "missing"), check.Equals, "missing")
}

/*
Test "installing Emacs on Debian not using yum" .
*/
func (s *CompilerTestSuite) TestDefinitionIncludePgSql(c *check.C) {
	c.Assert(len(s.cmp.Tree().GetBranch("state").GetList("install-postgres")), check.Equals, 2)
	//c.Log(s.cmp.Tree().ToYAML())
}
