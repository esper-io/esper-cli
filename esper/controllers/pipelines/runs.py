from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipelines_api import PipelinesApiAdapter
from esper.ext.utils import validate_creds_exists

class Runs(Controller):

    class Meta:
        label = 'runs'

        # text displayed at the top of --help output
        description = 'Pipeline runs'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli pipelines runs'

        stacked_type = 'nested'
        stacked_on = 'pipelines'

    @ex(
        help='Show runs for a pipeline',
        arguments=[
            (['-i', '--id'],
             {'help': 'run id',
              'action': 'store',
              'dest': 'run',
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

        run_id = self.app.pargs.run
        render_data = []

        if run_id:
            result = adapter.get_pipeline_run(pipeline['id'], run_id)
            render_data = [{
                'Run Number': result['content']['run_number'],
                'Status': result['content']['status'],
                'Id': run_id,
                'Created At': result['content']['created_at'],
            }]
        else:
            result = adapter.get_pipeline_runs(pipeline['id'])
            for run in result['content']['results']:
                run_render = {
                    'Run Number': run['run_number'],
                    'Status': run['status'],
                    'Id': run['id'],
                    'Created At': run['created_at'],
                    }
                render_data.append(run_render)

        self.app.render(f"Listing runs for the Pipeline! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Create a new run for the Pipeline',
        arguments=[]
    )
    def new(self):
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

        render_data = []
        result = adapter.create_pipeline_run(pipeline['id'])
        run = result['content']
        run_render = {
            'Run Number': run['run_number'],
            'Status': run['status'],
            'Id': run['id'],
            'Created At': run['created_at'],
            }
        render_data.append(run_render)

        self.app.render(f"Created new run for Pipeline! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Cancel a pipeline run',
        arguments=[
            (['-p', '--pipeline-run'],
             {'help': 'pipeline run id',
              'action': 'store',
              'dest': 'run',
              'default': None}),
        ]
    )
    def cancel(self):
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

        pipeline_run_id = self.app.pargs.run
        if not pipeline_run_id:
            pipeline_run_id = input("Id of the Pipeline run: ")

        cancel_data = {
            'status': 'CANCELLED'
        }

        render_data = []
        result = adapter.update_pipeline_run(pipeline['id'], pipeline_run_id, cancel_data)
        run = result['content']
        run_render = {
            'Run Number': run['run_number'],
            'Status': run['status'],
            'Id': run['id'],
            'Created At': run['created_at'],
            }
        render_data.append(run_render)

        self.app.render(f"Created new run for Pipeline! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
