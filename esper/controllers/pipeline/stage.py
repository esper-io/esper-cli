from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipeline_api import get_stage_url, create_stage, edit_stage, list_stages, delete_api,\
    APIException, render_single_dict
from esper.ext.utils import validate_creds_exists


class Stage(Controller):
    class Meta:
        label = 'stage'

        # text displayed at the top of --help output
        description = 'Pipeline Stage commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli pipeline stage'

        stacked_type = 'nested'
        stacked_on = 'pipeline'

    @ex(
        help='Add a Stage',
        arguments=[
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None}),
            (['-n', '--name'],
             {'help': 'Name for this Stage',
              'action': 'store',
              'dest': 'name',
              'default': None}),
            (['--desc'],
             {'help': 'Stage Description',
              'action': 'store',
              'dest': 'desc',
              'default': None}),
            (['-o', '--order'],
             {'help': 'Stage Ordering - This has to be unique within a pipeline',
              'action': 'store',
              'dest': 'order',
              'type': int,
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

        name = self.app.pargs.name
        if not name:
            name = input("Name of the Stage: ")

        order = self.app.pargs.order
        if not order:
            order = prompt.query("Order of this Stage: ")

        desc = self.app.pargs.desc
        if not desc:
            desc = input("Description for this Stage [optional]: ")

        # Calling Pipeline Graphs API
        url = get_stage_url(environment, enterprise_id, pipeline_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Creating Pipeline...")
            response = create_stage(url, api_key, name, order, desc)
        except APIException:
            self.app.render("ERROR in connecting to Environment!\n")
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
        data = render_single_dict(response.json())

        self.app.render(f"Added Stage to Pipeline Successfully! Details: \n")
        self.app.render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Edit a Stage',
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
            (['-n', '--name'],
             {'help': 'Name for this Stage',
              'action': 'store',
              'dest': 'name',
              'default': None}),
            (['--desc'],
             {'help': 'Stage Description',
              'action': 'store',
              'dest': 'desc',
              'default': None}),
            (['-o', '--order'],
             {'help': 'Stage Ordering - This has to be unique within a pipeline',
              'action': 'store',
              'dest': 'order',
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

        name = self.app.pargs.name
        if not name:
            name = input("Change the name of the Stage: ")

        desc = self.app.pargs.desc
        if not desc:
            desc = input("Change the description for this Stage [optional]: ")

        order = self.app.pargs.order
        if not order:
            order = input("Change the Ordering for this Stage [optional]: ")

        # Calling Pipeline Graphs API
        url = get_stage_url(environment, enterprise_id, pipeline_id=pipeline_id, stage_id=stage_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Editing Stage...")
            response = edit_stage(url, api_key, name, order, desc)
        except APIException:
            self.app.render("ERROR in connecting to Environment!\n")
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

        self.app.render(f"Edited Stage for this Pipeline Successfully! Details: \n")
        self.app.render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='List all Stages',
        arguments=[
            (['-p', '--pipeline-id'],
             {'help': 'Name of the Pipeline',
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

        # Calling Pipeline Graphs API
        url = get_stage_url(environment, enterprise_id, pipeline_id=pipeline_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Listing Stages...")
            response = list_stages(url, api_key)

        except APIException:
            self.app.render("ERROR in connecting to Environment!\n")
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
                "OPERATIONS": len(stage.get("operations")),
                "VERSION": stage.get("version")
            }
            render_data.append(render_pipeline)

        self.app.render(f"Listing Stages for the Pipeline! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Remove a Stage',
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

        # Calling Pipeline Graphs API
        url = get_stage_url(environment, enterprise_id, pipeline_id=pipeline_id, stage_id=stage_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Removing Stage...")
            response = delete_api(url, api_key)
        except APIException:
            self.app.render("ERROR in connecting to Environment!\n")
            return

        if not response.ok:
            self.app.log.debug(f"Response not OK. Status Code: {response.status_code}")
            self.app.log.debug(f"Response not OK. Response: {response.json()}")

            if response.status_code == 404:
                self.app.log.error("Stage not found!")

            if response.status_code == 500:
                self.app.log.error(f"Internal Server Error! {response.json()}")
            return

        self.app.render(f"Removed Stage for this Pipeline Successfully! \n")
