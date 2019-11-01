from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipeline_api import get_operation_url, create_operation, edit_operation, list_stages, delete_api, \
    APIException, render_single_dict
from esper.ext.utils import validate_creds_exists


class Operation(Controller):
    class Meta:
        label = 'operation'

        # text displayed at the top of --help output
        description = 'Pipeline Stage Operation commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli pipeline stage operation'

        stacked_type = 'nested'
        stacked_on = 'stage'

    @ex(
        help='Add a Stage',
        arguments=[
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None}),
            (['-s', '--stage-id'],
             {'help': 'Stage ID',
              'action': 'store',
              'dest': 'stage_id',
              'default': None}),
            (['-n', '--name'],
             {'help': 'Name for this Operation',
              'action': 'store',
              'dest': 'name',
              'default': None}),
            (['--desc'],
             {'help': 'Operation Description',
              'action': 'store',
              'dest': 'desc',
              'default': None}),
            (['-a, --action'],
             {'help': 'Action for this Operation',
              'action': 'store',
              'dest': 'action',
              'default': None})
        ]
    )
    def create(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        enterprise_id = db.get_enterprise_id()

        pipeline_id = self.app.pargs.pipeline_id
        if not pipeline_id:
            pipeline_id = prompt.query("Enter the Pipeline ID: ")

        stage_id = self.app.pargs.stage_id
        if not stage_id:
            stage_id = prompt.query("Enter the Stage ID: ")

        name = self.app.pargs.name
        if not name:
            name = input("Name of the Operation: ")

        action = self.app.pargs.action
        if not action:
            action = prompt.options(
                "Action for this Operation: ",
                options=[
                    {"selector": 1, "prompt": "App Install to a Group of Devices", "return": 1},
                    {"selector": 2, "prompt": "App UnInstall to a Group of Devices", "return": 2},
                    {"selector": 3, "prompt": "Reboot a Group of Devices", "return": 3},
                ])

        desc = self.app.pargs.desc
        if not desc:
            desc = input("Description for this Operation [optional]: ")

        # Calling Pipeline Graphs API
        url = get_operation_url(environment, enterprise_id, pipeline_id, stage_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Creating Operation...")
            response = create_operation(url, api_key, name, action, desc)
        except APIException:
            self.app.render("ERROR in connecting to Environment!")
            return

        if not response.ok:
            self.app.log.debug(f"Response not OK. Status Code: {response.status_code}")
            self.app.log.debug(f"Response not OK. Response: {response.json()}")
            if response.status_code == 400:
                errors = response.json().get('meta', {}).get('non_field_errors')
                if errors:
                    self.app.log.error(f"Validation Error: {errors}")
                if response.json().get("errors"):
                    if "The fields pipeline, ordering must make a unique set." in response.json().get("message"):
                        self.app.log.error(f"Operation Already created for this Stage!")
                    else:
                        self.app.log.error(f"Validation Error: {response.json().get('errors')}")

            if response.status_code == 404:
                self.app.log.error("Stage URL not found!")

            if response.status_code == 500:
                self.app.log.error(f"Internal Server Error! {response.json()}")
            return

        # Rendering table with populated values
        data = render_single_dict(response.json())

        self.app.render(f"Added Operation to Stage Successfully! Details: \n")
        self.app.render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Edit an Operation',
        arguments=[
            (['-s', '--stage-id'],
             {'help': 'Stage ID',
              'action': 'store',
              'dest': 'stage_id',
              'default': None}),
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None}),
            (['-o', '--operation-id'],
             {'help': 'Operation ID',
              'action': 'store',
              'dest': 'operation_id',
              'default': None}),
            (['-n', '--name'],
             {'help': 'Name for this Operation',
              'action': 'store',
              'dest': 'name',
              'default': None}),
            (['--desc'],
             {'help': 'Operation Description',
              'action': 'store',
              'dest': 'desc',
              'default': None}),
            (['-a', '--action'],
             {'help': 'Action for this Operation',
              'action': 'store',
              'dest': 'action',
              'type': int,
              'default': None})
        ]
    )
    def edit(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        enterprise_id = db.get_enterprise_id()

        pipeline_id = self.app.pargs.pipeline_id
        if not pipeline_id:
            pipeline_id = prompt.query("Enter the Pipeline ID: ")

        stage_id = self.app.pargs.stage_id
        if not stage_id:
            stage_id = prompt.query("Enter the Stage ID: ")

        operation_id = self.app.pargs.operation_id
        if not operation_id:
            operation_id = prompt.query("Enter the Operation ID: ")

        name = self.app.pargs.name
        if not name:
            name = input("Change the name of the Operation: ")

        desc = self.app.pargs.desc
        if not desc:
            desc = input("Change the description for this Operation [optional]: ")

        action = self.app.pargs.action
        if not action:
            action = prompt.options(
                "Action for this Operation: ",
                options=[
                    {"selector": 1, "prompt": "App Install to a Group of Devices", "return": 1},
                    {"selector": 2, "prompt": "App UnInstall to a Group of Devices", "return": 2},
                    {"selector": 3, "prompt": "Reboot a Group of Devices", "return": 3},
                ])

        # Calling Pipeline Graphs API
        url = get_operation_url(environment, enterprise_id, pipeline_id=pipeline_id, stage_id=stage_id, operation_id=operation_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Editing Operation...")
            response = edit_operation(url, api_key, name, action, desc)
        except APIException:
            self.app.render("ERROR in connecting to Environment!")
            return

        if not response.ok:
            self.app.log.debug(f"Response not OK. Status Code: {response.status_code}")
            self.app.log.debug(f"Response not OK. Response: {response.json()}")
            if response.status_code == 400:
                errors = response.json().get('meta', {}).get('non_field_errors')
                if errors:
                    self.app.log.error(f"Validation Error: {errors}")
                if response.json().get("errors"):
                    self.app.log.error(f"Validation Error: {response.json().get('errors')}")

            if response.status_code == 404:
                self.app.log.error("Pipeline URL not found!")

            if response.status_code == 500:
                self.app.log.error(f"Internal Server Error! {response.json()}")
            return

        # Rendering table with populated values
        data = render_single_dict(response.json())

        self.app.render(f"Edited Operation for this Stage Successfully! Details: \n")
        self.app.render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='List all Stages',
        arguments=[
            (['-s', '--stage-id'],
             {'help': 'Stage ID',
              'action': 'store',
              'dest': 'stage_id',
              'default': None}),
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None})
        ]
    )
    def show(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        enterprise_id = db.get_enterprise_id()

        pipeline_id = self.app.pargs.pipeline_id
        if not pipeline_id:
            pipeline_id = prompt.query("Enter the Pipeline ID: ")

        stage_id = self.app.pargs.stage_id
        if not stage_id:
            stage_id = prompt.query("Enter the Stage ID: ")

        # Calling Pipeline Graphs API
        url = get_operation_url(environment, enterprise_id, pipeline_id=pipeline_id, stage_id=stage_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Listing Operations...")
            response = list_stages(url, api_key)

        except APIException:
            self.app.render("ERROR in connecting to Environment!")
            return

        if not response.ok:
            self.app.log.debug(f"Response not OK. Status Code: {response.status_code}")
            self.app.log.debug(f"Response not OK. Response: {response.json()}")
            if response.status_code == 400:
                errors = response.json().get('meta', {}).get('non_field_errors')
                if errors:
                    self.app.log.error(f"Validation Error: {errors}")
                if response.json().get("errors"):
                    self.app.log.error(f"Validation Error: {response.json().get('errors')}")

            if response.status_code == 404:
                self.app.log.error("Stage URL not found!")

            if response.status_code == 500:
                self.app.log.error(f"Internal Server Error! {response.json()}")
            return

        # Rendering table with populated values
        data = response.json().get("results")

        render_data = []
        for stage in data:
            render_pipeline = {
                "ID": stage.get("id"),
                "NAME": stage.get("name"),
                "DESCRIPTION": stage.get("description"),
                "ORDERING": stage.get("ordering"),
                "ACTION": stage.get("action"),
                "VERSION": stage.get("version")
            }
            render_data.append(render_pipeline)

        self.app.render(f"Listing Operations for the Stage! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Remove an Operation',
        arguments=[
            (['-s', '--stage-id'],
             {'help': 'Stage ID',
              'action': 'store',
              'dest': 'stage_id',
              'default': None}),
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None}),
            (['-o', '--operation-id'],
             {'help': 'Operation ID',
              'action': 'store',
              'dest': 'operation_id',
              'default': None})
        ]
    )
    def remove(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        enterprise_id = db.get_enterprise_id()

        pipeline_id = self.app.pargs.pipeline_id
        if not pipeline_id:
            pipeline_id = prompt.query("Enter the Pipeline ID: ")

        stage_id = self.app.pargs.stage_id
        if not stage_id:
            stage_id = prompt.query("Enter the Stage ID: ")

        operation_id = self.app.pargs.operation_id
        if not operation_id:
            operation_id = prompt.query("Enter the Operation ID: ")

        # Calling Pipeline Graphs API
        url = get_operation_url(environment, enterprise_id, pipeline_id=pipeline_id, stage_id=stage_id,
                                operation_id=operation_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Removing Operation...")
            response = delete_api(url, api_key)
        except APIException:
            self.app.render("ERROR in connecting to Environment!")
            return

        if not response.ok:
            self.app.log.debug(f"Response not OK. Status Code: {response.status_code}")
            self.app.log.debug(f"Response not OK. Response: {response.json()}")

            if response.status_code == 404:
                self.app.log.error("Operation not found!")

            if response.status_code == 500:
                self.app.log.error(f"Internal Server Error! {response.json()}")
            return

        self.app.render(f"Removed Operation for this Stage Successfully! \n")
