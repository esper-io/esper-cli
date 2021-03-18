from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipelines_api import PipelinesApiAdapter
from esper.ext.utils import validate_creds_exists

class Operations(Controller):

    class Meta:
        label = 'operations'

        # text displayed at the top of --help output
        description = 'Pipeline operations'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli pipelines operations'

        stacked_type = 'nested'
        stacked_on = 'pipelines'

    @ex(
        help='Show operations for a stage',
        arguments=[
            (['-s', '--stage'],
             {'help': 'Stage to show operations for',
              'action': 'store',
              'dest': 'stage',
              'default': None}),
        ]
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

        stage_id = self.app.pargs.stage
        if not stage_id:
            stage_id = input("Id of the Stage: ")

        operationlists = adapter.get_stage_operationlists(stage_id)
        operationlist = operationlists['content']['results'][0]

        operations = adapter.get_operationlist_operations(operationlist['id'])
        operations = operations['content']['results']

        render_data = []
        for op in operations:
            op_render = {
                'Id': op['id'],
                'Type': op['operation_type'],
                'Args': op['args'],
            }
            render_data.append(op_render)

        self.app.render(f"Listing operations for the Stage! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
