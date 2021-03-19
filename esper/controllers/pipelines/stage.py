from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipelines_api import PipelinesApiAdapter
from esper.ext.utils import validate_creds_exists

class Stage(Controller):
    class Meta:
        label = 'stage'

        # text displayed at the top of --help output
        description = 'Pipeline Stage commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli pipelines stage'

        stacked_type = 'nested'
        stacked_on = 'pipelines'

    @ex(
        help='Add a Stage',
        arguments=[
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
        ]
    )
    def create(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        auth_token = db.get_auth_token()
        adapter = PipelinesApiAdapter(environment, auth_token['auth_token'])

        pipeline = db.get_pipeline()
        if pipeline is None or pipeline.get('id') is None:
            self.app.log.debug('[pipeline-active] There is no active pipeline.')
            self.app.render('There is no active pipeline. Please set an active pipeline before adding a Stage')
            return

        name = self.app.pargs.name
        if not name:
            name = input("Name of the Stage: ")

        desc = self.app.pargs.desc
        if not desc:
            desc = input("Description for this Stage [optional]: ")

        stage_data = {
            'name': name,
            'description': desc,
        }

        stage = adapter.create_stage(pipeline['id'], stage_data)
        stage = stage['content']

        # Rendering table with populated values
        stage_render = [{
            'Id': stage['id'],
            'Name': stage['name'],
            'Description': stage['description'],
            'Pipeline': stage['pipeline_id'],
            'Created At': stage['created_at'],
        }]

        self.app.render(f"Added Stage to Pipeline Successfully! Details: \n")
        self.app.render(stage_render, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='List all Stages of a Pipeline',
        arguments=[]
    )
    def show(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        auth_token = db.get_auth_token()
        adapter = PipelinesApiAdapter(environment, auth_token['auth_token'])

        pipeline = db.get_pipeline()
        if pipeline is None or pipeline.get('id') is None:
            self.app.log.debug('[pipeline-active] There is no active pipeline.')
            self.app.render('There is no active pipeline. Please set an active pipeline before getting a Stage')
            return

        stages = adapter.get_stages(pipeline['id'])['content']['results']

        render_data = []
        for stage in stages:
            stage_render = {
                "Id": stage['id'],
                "Name": stage["name"],
                "Description": stage["description"],
                "Pipeline": stage['pipeline_id'],
                "Created At": stage["created_at"],
            }
            render_data.append(stage_render)

        self.app.render(f"Listing Stages for the Pipeline! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Remove a Stage',
        arguments=[
            (['-s', '--stage-id'],
             {'help': 'Stage ID',
              'action': 'store',
              'dest': 'stage_id',
              'default': None})
        ]
    )
    def remove(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        auth_token = db.get_auth_token()
        adapter = PipelinesApiAdapter(environment, auth_token['auth_token'])

        stage_id = self.app.pargs.stage_id
        if not stage_id:
            self.app.render('Missing arguments! Provide a stage id to delete.')
            return

        pipeline = db.get_pipeline()
        if pipeline is None or pipeline.get('id') is None:
            self.app.log.debug('[pipeline-active] There is no active pipeline.')
            self.app.render('There is no active pipeline. Please set an active pipeline before getting a Stage')
            return

        adapter.delete_stage(pipeline['id'], stage_id)

        self.app.render(f"Deleted Stage Successfully! \n")
