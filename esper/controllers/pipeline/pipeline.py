from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipeline_api import get_pipeline_url, create_pipeline, edit_pipeline, list_pipelines, fetch_pipelines, APIException, render_single_dict
from esper.ext.utils import validate_creds_exists


class Pipeline(Controller):
    class Meta:
        label = 'pipeline'

        # text displayed at the top of --help output
        description = 'Pipeline commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli pipeline'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='Create a pipeline',
        arguments=[
            (['-n', '--name'],
             {'help': 'Name of the Pipeline',
              'action': 'store',
              'dest': 'name',
              'default': None}),
            (['--desc'],
             {'help': 'Pipeline Description',
              'action': 'store',
              'dest': 'desc',
              'default': None})
        ]
    )
    def create(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        enterprise_id = db.get_enterprise_id()

        name = self.app.pargs.name
        if not name:
            name = prompt.query("Name of the Pipeline: ")

        desc = self.app.pargs.desc
        if not desc:
            desc = input("Description for this Pipeline [optional]: ")

        # Calling Pipeline Graphs API
        url = get_pipeline_url(environment, enterprise_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Creating Pipeline...")
            response = create_pipeline(url, api_key, name, desc)
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

        self.app.render(f"Created Pipeline Successfully! Details: \n")
        self.app.render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Edit a pipeline(s)',
        arguments=[
            (['-n', '--name'],
             {'help': 'Name of the Pipeline',
              'action': 'store',
              'dest': 'name',
              'default': None}),
            (['--desc'],
             {'help': 'Pipeline Description',
              'action': 'store',
              'dest': 'desc',
              'default': None}),
            (['-p, --pipeline-id'],
             {'help': 'Pipeline ID',
              'action': 'store',
              'dest': 'pipeline_id',
              'default': None})
        ]
    )
    def edit(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        enterprise_id = db.get_enterprise_id()

        name = self.app.pargs.name
        if not name:
            name = input("Change the name of the Pipeline: ")

        desc = self.app.pargs.desc
        if not desc:
            desc = input("Change the description for this Pipeline [optional]: ")

        pipeline_id = self.app.pargs.pipeline_id
        if not pipeline_id:
            pipeline_id = prompt.query("Enter the Pipeline ID: ")

        if not name and not desc:
            self.app.render("No changes requested. Exiting app!")
            return

        # Calling Pipeline Graphs API
        url = get_pipeline_url(environment, enterprise_id, pipeline_id=pipeline_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Editing Pipeline...")
            response = edit_pipeline(url, api_key, name, desc)
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

        self.app.render(f"Edited Pipeline Successfully! Details: \n")
        self.app.render(data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='List or Fetch a pipeline(s)',
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

        # Calling Pipeline Graphs API
        url = get_pipeline_url(environment, enterprise_id, pipeline_id=pipeline_id)
        api_key = db.get_configure().get("api_key")

        try:
            self.app.log.debug("Listing Pipeline...")
            if pipeline_id:
                response = fetch_pipelines(url, api_key)
            else:
                response = list_pipelines(url, api_key)
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
        if not pipeline_id:
            data = response.json().get("results")
        else:
            data = [response.json()]

        render_data = []
        for pipeline in data:
            render_pipeline = {
                "ID": pipeline.get("id"),
                "NAME": pipeline.get("name"),
                "DESCRIPTION": pipeline.get("description"),
                "STAGES": len(pipeline.get("stages")),
                "VERSION": pipeline.get("version")
            }
            render_data.append(render_pipeline)

        self.app.render(f"Listing Pipeline! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")