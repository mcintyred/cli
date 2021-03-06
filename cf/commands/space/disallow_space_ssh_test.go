package space_test

import (
	"errors"

	"github.com/cloudfoundry/cli/cf/api/apifakes"
	"github.com/cloudfoundry/cli/cf/commandregistry"
	"github.com/cloudfoundry/cli/cf/configuration/coreconfig"
	"github.com/cloudfoundry/cli/cf/models"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"

	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("disallow-space-ssh command", func() {
	var (
		ui                  *testterm.FakeUI
		requirementsFactory *testreq.FakeReqFactory
		spaceRepo           *apifakes.FakeSpaceRepository
		configRepo          coreconfig.Repository
		deps                commandregistry.Dependency
	)

	BeforeEach(func() {
		ui = &testterm.FakeUI{}
		configRepo = testconfig.NewRepositoryWithDefaults()
		requirementsFactory = &testreq.FakeReqFactory{}
		spaceRepo = new(apifakes.FakeSpaceRepository)
	})

	updateCommandDependency := func(pluginCall bool) {
		deps.UI = ui
		deps.Config = configRepo
		deps.RepoLocator = deps.RepoLocator.SetSpaceRepository(spaceRepo)
		commandregistry.Commands.SetCommand(commandregistry.Commands.FindCommand("disallow-space-ssh").SetDependency(deps, pluginCall))
	}

	runCommand := func(args ...string) bool {
		return testcmd.RunCLICommand("disallow-space-ssh", args, requirementsFactory, updateCommandDependency, false)
	}

	Describe("requirements", func() {
		It("fails with usage when called without enough arguments", func() {
			requirementsFactory.LoginSuccess = true

			runCommand()
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Incorrect Usage", "Requires", "argument"},
			))

		})

		It("fails requirements when not logged in", func() {
			Expect(runCommand("my-space")).To(BeFalse())
		})

		It("does not pass requirements if org is not targeted", func() {
			requirementsFactory.TargetedOrgSuccess = false

			Expect(runCommand("my-space")).To(BeFalse())
		})

		It("does not pass requirements if space does not exist", func() {
			requirementsFactory.LoginSuccess = true
			requirementsFactory.TargetedOrgSuccess = true
			requirementsFactory.SpaceRequirementFails = true

			Expect(runCommand("my-space")).To(BeFalse())
		})
	})

	Describe("disallow-space-ssh", func() {
		var space models.Space

		BeforeEach(func() {
			requirementsFactory.LoginSuccess = true
			requirementsFactory.TargetedOrgSuccess = true

			space = models.Space{}
			space.Name = "the-space-name"
			space.GUID = "the-space-guid"
		})

		Context("when allow_ssh is already set to the false", func() {
			BeforeEach(func() {
				space.AllowSSH = false
				requirementsFactory.Space = space
			})

			It("notifies the user", func() {
				runCommand("the-space-name")

				Expect(ui.Outputs).To(ContainSubstrings([]string{"ssh support is already disabled in space 'the-space-name'"}))
			})
		})

		Context("Updating allow_ssh when not already set to false", func() {
			Context("Update successfully", func() {
				BeforeEach(func() {
					space.AllowSSH = true
					requirementsFactory.Space = space
				})

				It("updates the space's allow_ssh", func() {
					runCommand("the-space-name")

					Expect(spaceRepo.SetAllowSSHCallCount()).To(Equal(1))
					spaceGUID, allow := spaceRepo.SetAllowSSHArgsForCall(0)
					Expect(spaceGUID).To(Equal("the-space-guid"))
					Expect(allow).To(Equal(false))
					Expect(ui.Outputs).To(ContainSubstrings([]string{"Disabling ssh support for space 'the-space-name'"}))
					Expect(ui.Outputs).To(ContainSubstrings([]string{"OK"}))
				})
			})

			Context("Update fails", func() {
				BeforeEach(func() {
					space.AllowSSH = true
					requirementsFactory.Space = space
				})

				It("notifies user of any api error", func() {
					spaceRepo.SetAllowSSHReturns(errors.New("api error"))
					runCommand("the-space-name")

					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"FAILED"},
						[]string{"Error", "api error"},
					))

				})
			})

		})
	})

})
