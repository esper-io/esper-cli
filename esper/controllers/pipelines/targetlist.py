from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipelines_api import PipelinesApiAdapter
from esper.ext.utils import validate_creds_exists

class TargetList(Controller):
    class Meta:
        label = 'targetlist'

        # text displayed at the top of --help output
        description = 'Pipeline Target List commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli pipelines stage targetlist'

        stacked_type = 'nested'
        stacked_on = 'pipelines'

    @ex(
        help='Add a Stage Target list',
        arguments=[
            (['-n', '--name'],
             {'help': 'Name for this target list',
              'action': 'store',
              'dest': 'name',
              'default': None}),
            (['--t', '--type'],
             {'help': 'Target list type. DEVICES|GROUPS|COLLECTIONS',
              'action': 'store',
              'dest': 'targetlist_type',
              'default': None}),
            (['-s', '--stage'],
             {'help': 'Stage to attach this list to',
              'action': 'store',
              'dest': 'stage',
              'default': None}),
        ]
    )
    def create(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        api_key = db.get_configure().get("api_key")
        adapter = PipelinesApiAdapter(environment, api_key)

        pipeline = db.get_pipeline()
        if pipeline is None or pipeline.get('id') is None:
            self.app.log.debug('[pipeline-active] There is no active pipeline.')
            self.app.render('There is no active pipeline. Please set an active pipeline before adding a Stage')
            return

        name = self.app.pargs.name
        if not name:
            name = input("Name of the Target list: ")

        stage_id = self.app.pargs.stage
        if not stage_id:
            stage_id = input("Id of the Stage: ")

        targetlist_type = self.app.pargs.targetlist_type
        if not targetlist_type:
            targetlist_type = input("Type for this Target List: DEVICES|GROUPS|COLLECTIONS: ")

        targetlist_data = {
            'name': name,
            'target_list_type': targetlist_type,
        }

        targetlist = adapter.create_targetlist(pipeline['id'], targetlist_data)
        targetlist = targetlist['content']

        targetlist = adapter.add_stage_target_list(stage_id, targetlist['id'])
        targetlist = targetlist['content']

        # Rendering table with populated values
        targetlist_render = [{
            'Id': targetlist['id'],
            'Pipeline': targetlist['pipeline_id'],
            'Name': targetlist['name'],
            'Created At': targetlist['created_at'],
        }]

        self.app.render(f"Added Target list to Stage Successfully! Details: \n")
        self.app.render(targetlist_render, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='List Target lists of a Stage',
        arguments=[
            (['-s', '--stage'],
             {'help': 'Stage to attach this list to',
              'action': 'store',
              'dest': 'stage',
              'default': None}),
        ]
    )
    def show(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        api_key = db.get_configure().get("api_key")
        adapter = PipelinesApiAdapter(environment, api_key)

        pipeline = db.get_pipeline()
        if pipeline is None or pipeline.get('id') is None:
            self.app.log.debug('[pipeline-active] There is no active pipeline.')
            self.app.render('There is no active pipeline. Please set an active pipeline before getting a Stage')
            return

        stage_id = self.app.pargs.stage
        if not stage_id:
            stage_id = input("Id of the Stage: ")

        targetlists = adapter.get_stage_target_lists(stage_id)
        targetlists = targetlists['content']['results']

        render_data = []
        for targetlist in targetlists:
            targetlist_render = {
                'Id': targetlist['id'],
                'Pipeline': targetlist['pipeline_id'],
                'Name': targetlist['name'],
                'Created At': targetlist['created_at'],
            }
            render_data.append(targetlist_render)

        self.app.render(f"Listing targetlists for the Stage! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
