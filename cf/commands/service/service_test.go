package service_test

import (
	"github.com/cloudfoundry/cli/cf/commandregistry"
	"github.com/cloudfoundry/cli/cf/models"
	"github.com/cloudfoundry/cli/cf/trace/tracefakes"
	"github.com/cloudfoundry/cli/plugin/models"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"

	. "github.com/cloudfoundry/cli/cf/commands/service"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("service command", func() {
	var (
		ui                  *testterm.FakeUI
		requirementsFactory *testreq.FakeReqFactory
		deps                commandregistry.Dependency
	)

	updateCommandDependency := func(pluginCall bool) {
		deps.UI = ui
		commandregistry.Commands.SetCommand(commandregistry.Commands.FindCommand("service").SetDependency(deps, pluginCall))
	}

	BeforeEach(func() {
		ui = &testterm.FakeUI{}
		requirementsFactory = &testreq.FakeReqFactory{}

		deps = commandregistry.NewDependency(new(tracefakes.FakePrinter))
	})

	runCommand := func(args ...string) bool {
		return testcmd.RunCLICommand("service", args, requirementsFactory, updateCommandDependency, false)
	}

	Describe("requirements", func() {
		It("fails when not provided the name of the service to show", func() {
			requirementsFactory.LoginSuccess = true
			requirementsFactory.TargetedSpaceSuccess = true
			runCommand()

			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Incorrect Usage", "Requires an argument"},
			))
		})

		It("fails when not logged in", func() {
			requirementsFactory.TargetedSpaceSuccess = true

			Expect(runCommand("come-ON")).To(BeFalse())
		})

		It("fails when a space is not targeted", func() {
			requirementsFactory.LoginSuccess = true

			Expect(runCommand("okay-this-time-please??")).To(BeFalse())
		})
	})

	Describe("After Requirement", func() {
		createServiceInstanceWithState := func(state string) {
			offering := models.ServiceOfferingFields{Label: "mysql", DocumentationURL: "http://documentation.url", Description: "the-description"}
			plan := models.ServicePlanFields{GUID: "plan-guid", Name: "plan-name"}

			serviceInstance := models.ServiceInstance{}
			serviceInstance.Name = "service1"
			serviceInstance.GUID = "service1-guid"
			serviceInstance.LastOperation.Type = "create"
			serviceInstance.LastOperation.State = "in progress"
			serviceInstance.LastOperation.Description = "creating resource - step 1"
			serviceInstance.ServicePlan = plan
			serviceInstance.ServiceOffering = offering
			serviceInstance.DashboardURL = "some-url"
			serviceInstance.LastOperation.State = state
			serviceInstance.LastOperation.CreatedAt = "created-date"
			serviceInstance.LastOperation.UpdatedAt = "updated-date"
			requirementsFactory.ServiceInstance = serviceInstance
		}

		createServiceInstance := func() {
			createServiceInstanceWithState("")
		}

		Describe("when invoked by a plugin", func() {
			var (
				pluginModel *plugin_models.GetService_Model
			)

			BeforeEach(func() {
				requirementsFactory.LoginSuccess = true
				requirementsFactory.TargetedSpaceSuccess = true

				pluginModel = &plugin_models.GetService_Model{}
				deps.PluginModels.Service = pluginModel
			})

			It("populates the plugin model upon execution", func() {
				createServiceInstanceWithState("in progress")
				testcmd.RunCLICommand("service", []string{"service1"}, requirementsFactory, updateCommandDependency, true)
				Expect(pluginModel.Name).To(Equal("service1"))
				Expect(pluginModel.Guid).To(Equal("service1-guid"))
				Expect(pluginModel.LastOperation.Type).To(Equal("create"))
				Expect(pluginModel.LastOperation.State).To(Equal("in progress"))
				Expect(pluginModel.LastOperation.Description).To(Equal("creating resource - step 1"))
				Expect(pluginModel.LastOperation.CreatedAt).To(Equal("created-date"))
				Expect(pluginModel.LastOperation.UpdatedAt).To(Equal("updated-date"))
				Expect(pluginModel.LastOperation.Type).To(Equal("create"))
				Expect(pluginModel.ServicePlan.Name).To(Equal("plan-name"))
				Expect(pluginModel.ServicePlan.Guid).To(Equal("plan-guid"))
				Expect(pluginModel.ServiceOffering.DocumentationUrl).To(Equal("http://documentation.url"))
				Expect(pluginModel.ServiceOffering.Name).To(Equal("mysql"))
			})
		})

		Context("when logged in, a space is targeted, and provided the name of a service that exists", func() {
			BeforeEach(func() {
				requirementsFactory.LoginSuccess = true
				requirementsFactory.TargetedSpaceSuccess = true
			})

			Context("when the service is externally provided", func() {

				It("shows the service", func() {
					createServiceInstanceWithState("in progress")
					runCommand("service1")

					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"Service instance:", "service1"},
						[]string{"Service: ", "mysql"},
						[]string{"Plan: ", "plan-name"},
						[]string{"Description: ", "the-description"},
						[]string{"Documentation url: ", "http://documentation.url"},
						[]string{"Dashboard: ", "some-url"},
						[]string{"Last Operation"},
						[]string{"Status: ", "create in progress"},
						[]string{"Message: ", "creating resource - step 1"},
						[]string{"Started: ", "created-date"},
						[]string{"Updated: ", "updated-date"},
					))
					Expect(requirementsFactory.ServiceInstanceName).To(Equal("service1"))
				})

				Context("when the service instance CreatedAt is empty", func() {
					It("does not output the Started line", func() {
						createServiceInstanceWithState("in progress")
						requirementsFactory.ServiceInstance.LastOperation.CreatedAt = ""
						runCommand("service1")

						Expect(ui.Outputs).To(ContainSubstrings(
							[]string{"Service instance:", "service1"},
							[]string{"Service: ", "mysql"},
							[]string{"Plan: ", "plan-name"},
							[]string{"Description: ", "the-description"},
							[]string{"Documentation url: ", "http://documentation.url"},
							[]string{"Dashboard: ", "some-url"},
							[]string{"Last Operation"},
							[]string{"Status: ", "create in progress"},
							[]string{"Message: ", "creating resource - step 1"},
							[]string{"Updated: ", "updated-date"},
						))
						Expect(ui.Outputs).ToNot(ContainSubstrings(
							[]string{"Started: "},
						))
					})
				})

				Context("shows correct status information based on service instance state", func() {
					It("shows status: `create in progress` when state is `in progress`", func() {
						createServiceInstanceWithState("in progress")
						runCommand("service1")

						Expect(ui.Outputs).To(ContainSubstrings(
							[]string{"Status: ", "create in progress"},
						))
						Expect(requirementsFactory.ServiceInstanceName).To(Equal("service1"))
					})

					It("shows status: `create succeeded` when state is `succeeded`", func() {
						createServiceInstanceWithState("succeeded")
						runCommand("service1")

						Expect(ui.Outputs).To(ContainSubstrings(
							[]string{"Status: ", "create succeeded"},
						))
						Expect(requirementsFactory.ServiceInstanceName).To(Equal("service1"))
					})

					It("shows status: `create failed` when state is `failed`", func() {
						createServiceInstanceWithState("failed")
						runCommand("service1")

						Expect(ui.Outputs).To(ContainSubstrings(
							[]string{"Status: ", "create failed"},
						))
						Expect(requirementsFactory.ServiceInstanceName).To(Equal("service1"))
					})

					It("shows status: `` when state is ``", func() {
						createServiceInstanceWithState("")
						runCommand("service1")

						Expect(ui.Outputs).To(ContainSubstrings(
							[]string{"Status: ", ""},
						))
						Expect(requirementsFactory.ServiceInstanceName).To(Equal("service1"))
					})
				})

				Context("when the guid flag is provided", func() {
					It("shows only the service guid", func() {
						createServiceInstance()
						runCommand("--guid", "service1")

						Expect(ui.Outputs).To(ContainSubstrings(
							[]string{"service1-guid"},
						))

						Expect(ui.Outputs).ToNot(ContainSubstrings(
							[]string{"Service instance:", "service1"},
						))
					})
				})
			})

			Context("when the service is user provided", func() {
				BeforeEach(func() {
					serviceInstance := models.ServiceInstance{}
					serviceInstance.Name = "service1"
					serviceInstance.GUID = "service1-guid"
					requirementsFactory.ServiceInstance = serviceInstance
				})

				It("shows user provided services", func() {
					runCommand("service1")

					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"Service instance: ", "service1"},
						[]string{"Service: ", "user-provided"},
					))
				})
			})

			Context("when the service has tags", func() {
				BeforeEach(func() {
					serviceInstance := models.ServiceInstance{}
					serviceInstance.Tags = []string{"tag1", "tag2"}
					serviceInstance.ServicePlan = models.ServicePlanFields{GUID: "plan-guid", Name: "plan-name"}
					requirementsFactory.ServiceInstance = serviceInstance
				})

				It("includes the tags in the output", func() {
					runCommand("service1")

					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"Tags: ", "tag1, tag2"},
					))
				})
			})
		})
	})
})

var _ = Describe("ServiceInstanceStateToStatus", func() {
	var operationType string
	Context("when the service is not user provided", func() {
		isUserProvided := false

		Context("when operationType is `create`", func() {
			BeforeEach(func() { operationType = "create" })

			It("returns status: `create in progress` when state: `in progress`", func() {
				status := ServiceInstanceStateToStatus(operationType, "in progress", isUserProvided)
				Expect(status).To(Equal("create in progress"))
			})

			It("returns status: `create succeeded` when state: `succeeded`", func() {
				status := ServiceInstanceStateToStatus(operationType, "succeeded", isUserProvided)
				Expect(status).To(Equal("create succeeded"))
			})

			It("returns status: `create failed` when state: `failed`", func() {
				status := ServiceInstanceStateToStatus(operationType, "failed", isUserProvided)
				Expect(status).To(Equal("create failed"))
			})

			It("returns status: `` when state: ``", func() {
				status := ServiceInstanceStateToStatus(operationType, "", isUserProvided)
				Expect(status).To(Equal(""))
			})
		})
	})

	Context("when the service is user provided", func() {
		isUserProvided := true

		It("returns status: `` when state: ``", func() {
			status := ServiceInstanceStateToStatus(operationType, "", isUserProvided)
			Expect(status).To(Equal(""))
		})
	})
})
