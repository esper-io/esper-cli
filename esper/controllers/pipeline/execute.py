from enum import Enum

from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipeline_api import execute_pipeline, list_execute_pipeline, get_pipeline_execute_url, \
    APIException, render_single_dict
from esper.ext.utils import validate_creds_exists


class ActionEnums(Enum):
    APP_INSTALL = "App Install to a Group of Devices"
    APP_UNINSTALL = "App Uninstall to a Group of Devices"
    REBOOT = "Reboot a Group of Devices"

    @classmethod
    def choices_values(cls):
        return [member.value for _, member in cls.__members__.items()]


class Execution(Controller):
    class Meta:
        label = 'execute'

        # text displayed at the top of --help output
        description = 'Pipeline Execute commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli pipeline execute'

        stacked_type = 'nested'
        stacked_on = 'pipeline'

    def fetch_args(self):
        results = {}

        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        results["environment"] = db.get_configure().get("environment")
        results["enterprise_id"] = db.get_enterprise_id()
        results["api_key"] = db.get_configure().get("api_key")

        if hasattr(self.app.pargs, "pipeline_id"):
            pipeline_id = self.app.pargs.pipeline_id
            if not pipeline_id:
                pipeline_id = prompt.query("Enter the Pipeline ID: ")

            results["pipeline_id"] = pipeline_id

        if hasattr(self.app.pargs, "execution_id"):
            execution_id = self.app.pargs.execution_id
            if not execution_id:
                execution_id = prompt.query("Enter the Execution ID: ")

            results["execution_id"] = execution_id

        if hasattr(self.app.pargs, "reason"):
            reason = self.app.pargs.reason
            if not reason:
                reason = prompt.query("Why do you want to stop this Execution? : ")

            results["reason"] = reason

        return results

    def handle_response_failure(self, response):
        self.app.log.debug(f"Response not OK. Status Code: {response.status_code}")
        self.app.log.debug(f"Response not OK. Response: {response.json()}")
        if response.status_code == 400:

            if isinstance(response.json(), dict):
                errors = response.json().get('meta', {}).get('non_field_errors')
                if errors:
                    self.app.log.error(f"Validation Error: {errors}")
                if response.json().get("errors"):
                    if "The fields pipeline, ordering must make a unique set." in response.json().get("message"):
                        self.app.log.error(f"Operation with same `name` already created for this Stage!")
                    else:
                        self.app.log.error(f"Validation Error: {response.json().get('errors')}")
            else:
                self.app.log.error(f"Validation Errors -> {response.json()}")

        if response.status_code == 404:
            self.app.log.debug(f"Pipeline URL not found! URL -> {response.url}")
            self.app.log.error("Pipeline URL not found!")

        if response.status_code == 500:
            self.app.log.error(f"Internal Server Error! {response.json()}")

    @ex(
        help='Execute pipeline',
        arguments=[
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None}),
        ]
    )
    def start(self):
        args = self.fetch_args()

        # Calling Pipeline Execute API
        url = get_pipeline_execute_url(args["environment"], args["enterprise_id"], args["pipeline_id"])

        try:
            self.app.log.debug("Executing Pipeline...")
            response = execute_pipeline(url, args["api_key"])

        except APIException:
            self.app.render("ERROR in connecting to Environment!\n")
            return

        if not response.ok:
            self.handle_response_failure(response)
            return

        # Rendering table with populated values
        data = render_single_dict(response.json())

        self.app.render(f"Pipeline execution started! Details: \n")
        self.app.render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Stop a Pipeline Execution',
        arguments=[
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None}),
            (['-e', '--execution-id'],
             {'help': 'Execution ID',
              'action': 'store',
              'dest': 'execution_id',
              'default': None}),
            (['-r', '--reason'],
             {'help': 'Reason for stopping Execution',
              'action': 'store',
              'dest': 'reason',
              'default': None})
        ]
    )
    def stop(self):
        args = self.fetch_args()

        data = {
            "reason": args["reason"]
        }

        # Calling Pipeline Execute API
        url = get_pipeline_execute_url(
            args["environment"],
            args["enterprise_id"],
            args["pipeline_id"],
            args["execution_id"],
            "stop"
        )

        try:
            self.app.log.debug("Stopping Pipeline Execution...")
            response = execute_pipeline(url, args["api_key"], data)
        except APIException:
            self.app.render("ERROR in connecting to Environment!\n")
            return

        if not response.ok:
            self.handle_response_failure(response)
            return

        # Rendering table with populated values
        data = render_single_dict(response.json())

        self.app.render(f"Pipeline execution stopped! Details: \n")
        self.app.render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Continue a Pipeline Execution',
        label="continue",
        arguments=[
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None}),
            (['-e', '--execution-id'],
             {'help': 'Execution ID',
              'action': 'store',
              'dest': 'execution_id',
              'default': None})
        ]
    )
    def ex_continue(self):
        args = self.fetch_args()

        # Calling Pipeline Execute API
        url = get_pipeline_execute_url(
            args["environment"],
            args["enterprise_id"],
            args["pipeline_id"],
            args["execution_id"],
            "continue"
        )

        try:
            self.app.log.debug("Continuing Pipeline Execution...")
            response = execute_pipeline(url, args["api_key"])
        except APIException:
            self.app.render("ERROR in connecting to Environment!")
            return

        if not response.ok:
            self.handle_response_failure(response)
            return

        # Rendering table with populated values
        data = render_single_dict(response.json())

        self.app.render(f"Pipeline execution continuing! Details: \n")
        self.app.render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Terminate a Pipeline Execution',
        arguments=[
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None}),
            (['-e', '--execution-id'],
             {'help': 'Execution ID',
              'action': 'store',
              'dest': 'execution_id',
              'default': None}),
            (['-r', '--reason'],
             {'help': 'Reason for Terminating Execution',
              'action': 'store',
              'dest': 'reason',
              'default': None})
        ]
    )
    def terminate(self):
        args = self.fetch_args()

        data = {
            "reason": args["reason"]
        }

        # Calling Pipeline Execute API
        url = get_pipeline_execute_url(
            args["environment"],
            args["enterprise_id"],
            args["pipeline_id"],
            args["execution_id"],
            "terminate"
        )

        try:
            self.app.log.debug("Terminating Pipeline Execution...")
            response = execute_pipeline(url, args["api_key"], data)
        except APIException:
            self.app.render("ERROR in connecting to Environment!\n")
            return

        if not response.ok:
            self.handle_response_failure(response)
            return

        # Rendering table with populated values
        data = render_single_dict(response.json())

        self.app.render(f"Pipeline execution Terminated! Details: \n")
        self.app.render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='List all Executions',
        arguments=[
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None})
        ]
    )
    def show(self):
        args = self.fetch_args()

        # Calling Pipeline Graphs API
        url = get_pipeline_execute_url(
            args["environment"],
            args["enterprise_id"],
            args["pipeline_id"],
            execute_id=args.get("execution_id")
        )

        try:
            self.app.log.debug("Listing Executions...")
            response = list_execute_pipeline(url, args["api_key"])

        except APIException:
            self.app.render("ERROR in connecting to Environment!\n")
            return

        if not response.ok:
            self.handle_response_failure(response)
            return

        # Rendering table with populated values
        data = response.json().get("results")

        render_data = []
        for execution in data:
            render_pipeline = {
                "ID": execution.get("id"),
                "NAME": execution.get("name"),
                "DESCRIPTION": execution.get("description"),
                "STATE": execution.get("state"),
                "STATUS": execution.get("status"),
                "REASON": execution.get("reason"),
            }
            render_data.append(render_pipeline)

        self.app.render(f"Listing Executions for the Pipeline! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
