from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists
from esper.ext.pipelines_api import PipelinesApiAdapter

class Pipelines(Controller):
    class Meta:
        label = 'pipelines'

        # text displayed at the top of --help output
        description = 'Pipelines commands'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli pipelines'

        stacked_type = 'nested'
        stacked_on = 'base'

    def _configure_auth_token(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        auth_token = db.get_auth_token()
        if auth_token is None or auth_token.get('auth_token') is None:
            self.app.log.debug('[pipeline-token] There is no auth token.')

            new_auth_token = prompt.query("Auth token Id: ")
            db.set_auth_token(
                {
                    'auth_token': new_auth_token,
                }
            )

    @ex(
        help='Create a pipeline',
        arguments=[
            (['-n', '--name'],
             {'help': 'Name of the Pipeline',
              'action': 'store',
              'dest': 'name',
              'default': None}),
            (['--description'],
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
        self._configure_auth_token()
        auth_token = db.get_auth_token()
        adapter = PipelinesApiAdapter(environment, auth_token['auth_token'])

        name = self.app.pargs.name
        if not name:
            name = prompt.query("Name of the Pipeline: ")

        desc = self.app.pargs.desc
        if not desc:
            desc = input("Description for this Pipeline [optional]: ")

        pipelines_data = {
            'name': name,
            'description': desc,
        }

        pipeline = adapter.create_pipeline(pipelines_data)
        pipeline = pipeline['content']

        pipeline_render = [{
            'Id': pipeline['id'],
            'Name': pipeline['name'],
            'Description': pipeline['description'],
            'Created At': pipeline['created_at'],
        }]


        self.app.render(f"Created Pipeline Successfully! Details: \n")
        self.app.render(pipeline_render, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

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
        self._configure_auth_token()
        auth_token = db.get_auth_token()
        adapter = PipelinesApiAdapter(environment, auth_token['auth_token'])

        pipeline_id = self.app.pargs.pipeline_id

        pipelines_list = []
        if pipeline_id:
            pipeline = adapter.get_pipeline(pipeline_id)
            pipelines_list.append(pipeline['content'])
        else:
            pipelines_list = adapter.get_pipelines()['content']['results']

        render_data = []
        for pipeline in pipelines_list:
            pipeline_render = {
                'Id': pipeline['id'],
                'Name': pipeline['name'],
                'Description': pipeline['description'],
                'Created At': pipeline['created_at'],
            }
            render_data.append(pipeline_render)

        self.app.render(f"Listing Pipeline! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Set or show the active Pipeline',
        arguments=[
            (['-p', '--pipeline-id'],
             {'help': 'Pipeline id.',
              'action': 'store',
              'dest': 'pipeline_id'}),
        ]
    )
    def set_active(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        self._configure_auth_token()
        auth_token = db.get_auth_token()
        adapter = PipelinesApiAdapter(environment, auth_token['auth_token'])

        pipeline_id = self.app.pargs.pipeline_id
        if pipeline_id:
            pipeline = adapter.get_pipeline(pipeline_id)
            if not pipeline:
                self.app.render(f'Pipeline does not exist with ID {pipeline_id} \n')
                return

            db.set_pipeline(
                {
                    'id': pipeline['content']['id'],
                    'name': pipeline['content']['name'],
                }
            )
        else:
            pipeline = db.get_pipeline()
            if pipeline is None or pipeline.get('id') is None:
                self.app.log.debug('[pipeline-active] There is no active pipeline.')
                self.app.render('There is no active pipeline.')
                return

            pipeline_id = pipeline.get('id')

        self.app.render(f'Active Pipeline ID {pipeline_id} \n')

    @ex(
        help='Unset the current active pipeline',
        arguments=[]
    )
    def unset_active(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)

        pipeline = db.get_pipeline()
        if pipeline is None or pipeline.get('id') is None:
            self.app.log.debug('[pipeline-active] There is no active pipeline.')
            self.app.render('There is no active pipeline.')
            return

        db.unset_pipeline()
        self.app.render(f"Unset the active pipeline \n")

    @ex(
        help='Remove a Pipeline',
        arguments=[
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
        self._configure_auth_token()
        auth_token = db.get_auth_token()
        adapter = PipelinesApiAdapter(environment, auth_token['auth_token'])

        pipeline_id = self.app.pargs.pipeline_id
        if not pipeline_id:
            self.app.render('Missing arguments! Provide a pipeline id to delete.')
            return

        adapter.delete_pipeline(pipeline_id)

        self.app.render(f"Removed Pipeline Successfully! \n")
