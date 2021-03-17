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

        api_key = db.get_configure().get("api_key")


        data = {
            'foo': 'bar'
        }

        self.app.render(f"Created Pipeline Successfully! Details: \n")
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

        pipeline_id = self.app.pargs.pipeline_id

        api_key = db.get_configure().get("api_key")
        adapter = PipelinesApiAdapter(environment, api_key)

        pipelines_list = adapter.get_pipelines()

        render_data = []
        for pipeline in pipelines_list['content']['results']:
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
        enterprise_id = db.get_enterprise_id()

        pipeline_id = self.app.pargs.pipeline_id
        if not pipeline_id:
            pipeline_id = prompt.query("Enter the Pipeline ID: ")

        self.app.render(f"Removed Pipeline Successfully! \n")
