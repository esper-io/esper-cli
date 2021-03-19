from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipelines_api import PipelinesApiAdapter
from esper.ext.utils import validate_creds_exists

class TargetRuns(Controller):

    class Meta:
        label = 'targetruns'

        # text displayed at the top of --help output
        description = 'Target runs'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli pipelines runs targetruns'

        stacked_type = 'nested'
        stacked_on = 'stageruns'

    @ex(
        help='Show stage runs for a pipeline run',
        arguments=[
            (['-s', '--stage-run-id'],
             {'help': 'stage run id',
              'action': 'store',
              'dest': 'stagerun',
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

        stage_run_id = self.app.pargs.stagerun
        render_data = []

        if not stage_run_id:
            stage_run_id = input("Id of the Stage run: ")

        result = adapter.get_stage_run_target_runs(stage_run_id)
        for target_run in result:
            target_run_render = {
                'Target Id': target_run['target_id'],
                'Status': target_run['status'],
                'Command Status': target_run['command_status'],
                'Command Info': target_run['command_meta'],
                'Created At': target_run['created_at'],
                }
            render_data.append(target_run_render)

        self.app.render(f"Listing target runs for the Stage! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
