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
                'Pkg Name': op['args']['package_name'],
                'Version': op['args']['version_name'],
                'Version Code': op['args']['version_code'],
            }
            render_data.append(op_render)

        self.app.render(f"Listing operations for the Stage! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Create and operation for a stage',
        arguments=[
            (['-s', '--stage'],
             {'help': 'Stage to create operations for',
              'action': 'store',
              'dest': 'stage',
              'default': None}),
            (['-p', '--pkg-name'],
             {'help': 'App package name to install',
              'action': 'store',
              'dest': 'pkg_name',
              'default': None}),
            (['-a', '--app-version'],
             {'help': 'App version to install',
              'action': 'store',
              'dest': 'app_version',
              'default': None}),
            (['-v', '--version-code'],
             {'help': 'App version code',
              'action': 'store',
              'dest': 'version_code',
              'default': None}),
            (['--version-name'],
             {'help': 'App version name',
              'action': 'store',
              'dest': 'version_name',
              'default': None}),
            (['--application-name'],
             {'help': 'Application name',
              'action': 'store',
              'dest': 'app_name',
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
            self.app.render('There is no active pipeline. Please set an active pipeline before getting a Stage')
            return

        stage_id = self.app.pargs.stage
        if not stage_id:
            stage_id = input("Id of the Stage: ")

        pkg_name = self.app.pargs.pkg_name
        if not pkg_name:
            pkg_name = input("Package name e.g. com.acme.roadrunner: ")

        app_version = self.app.pargs.app_version
        if not app_version:
            app_version = input("App Version e.g. 1.2.3: ")

        version_code = self.app.pargs.version_code
        if not version_code:
            version_code = input("App Version Code e.g. 13253: ")

        version_name = self.app.pargs.version_name
        if not version_name:
            version_name = input("App Version Name(optional) e.g. 2.3.4: ")

        app_name = self.app.pargs.app_name
        if not app_name:
            app_name = input("App Name(optional) e.g. Road Runner: ")

        operationlists = adapter.get_stage_operationlists(stage_id)
        operationlist = operationlists['content']['results'][0]
        if not operationlist:
            # create operation list
            operationlist_data = {
                'name': 'op-list-name',
                'description': 'desc',
                'pipeline_id': pipeline['id'],
            }
            operationlist = adapter.create_stage_operationlist(stage_id, operationlist_data)
            operationlist = operationlist['content']

        operations = adapter.get_operationlist_operations(operationlist['id'])
        if operations['content']['count'] > 0:
            operation = operations['content']['results'][0]
            adapter.delete_operationlist_operation(operationlist['id'], operation['id'])

        install_op_data = {
            'args': {
                'app_version': app_version,
                'package_name': pkg_name,
                'version_code': version_code,
                'version_name': version_name,
                'application_name': app_name,

            },
            'operation_type': 'INSTALL'
        }

        operation = adapter.create_operationlist_operation(operationlist['id'], install_op_data)
        operation = operation['content']

        render_data = []
        op_render = {
            'Id': operation['id'],
            'Type': operation['operation_type'],
            'Pkg Name': operation['args']['package_name'],
            'Version': operation['args']['version_name'],
            'Version Code': operation['args']['version_code'],
        }
        render_data.append(op_render)

        self.app.render(f"Listing new operation for the Stage! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
