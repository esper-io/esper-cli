from cement import Controller, ex
from clint.textui import prompt

from esper.controllers.enums import OutputFormat
from esper.ext.db_wrapper import DBWrapper
from esper.ext.pipelines_api import PipelinesApiAdapter
from esper.ext.utils import validate_creds_exists

class TargetList(Controller):
    # TODO
    # - add "targetlist add --type = device|group|collection, --id = <id>"
    #   - should either create a target, targetlistdevicegroup, or targetlistcollection
    # - update show command to show the devices/groups/collections in the target list

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
        auth_token = db.get_auth_token()
        adapter = PipelinesApiAdapter(environment, auth_token['auth_token'])

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
            'Stage': stage_id,
            'Name': targetlist['name'],
            'Created At': targetlist['created_at'],
        }]

        self.app.render(f"Added Target list to Stage Successfully! Details: \n")
        self.app.render(targetlist_render, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Add contents Target list',
        arguments=[
            (['-t', '--targetlist-id'],
             {'help': 'Targetlist ID',
              'action': 'store',
              'dest': 'targetlist_id',
              'default': None}),
            (['-i', '--id'],
             {'help': 'ID of the device, group or collection',
              'action': 'store',
              'dest': 'content_id',
              'default': None}),
        ]
    )
    def add(self):
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

        targetlist_id = self.app.pargs.targetlist_id
        if not targetlist_id:
            self.app.render('Missing arguments! Provide a targetlist id to delete. ')
            targetlist_id = input("Id of the Targetlist: ")

        content_id = self.app.pargs.content_id
        if not content_id:
            content_id = input("ID of content to add to the list: ")

        targetlist = adapter.get_targetlist(pipeline['id'], targetlist_id)
        targetlist = targetlist['content']

        targetlist_content = None
        targetlist_type = targetlist['target_list_type']
        if targetlist_type == 'DEVICES':
            targetlist_content = {
                'device_id': content_id,
            }
            adapter.add_targetlist_device(pipeline['id'], targetlist_id, targetlist_content)
        elif targetlist_type == 'GROUPS':
            targetlist_content = {
                'device_group_id': content_id,
            }
            adapter.add_targetlist_device_group(pipeline['id'], targetlist_id, targetlist_content)
        elif targetlist_type == 'COLLECTIONS':
            targetlist_content = {
                'collection_id': content_id,
            }
            self.app.render('Collections not supported, cannot add {0} to target list'.format(content_id))
        else:
            self.app.render('Invalid target list type, cannot add {0} to target list'.format(content_id))
            return

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

        targetlists = adapter.get_stage_target_lists(stage_id)
        targetlists = targetlists['content']['results']

        render_data = []
        for targetlist in targetlists:
            targetlist_render = {
                'Id': targetlist['id'],
                'Stage': stage_id,
                'Name': targetlist['name'],
                'Created At': targetlist['created_at'],
            }
            render_data.append(targetlist_render)

        self.app.render(f"Listing targetlists for the Stage! Details: \n")
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Show contents of target list',
        arguments=[
            (['-s', '--stage'],
             {'help': 'Stage to attach this list to',
              'action': 'store',
              'dest': 'stage',
              'default': None}),
            (['-t', '--targetlist'],
             {'help': 'Stage to attach this list to',
              'action': 'store',
              'dest': 'targetlist_id',
              'default': None}),
        ]
    )
    def contents(self):
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

        targetlist_id = self.app.pargs.targetlist_id
        if not targetlist_id:
            targetlist_id = input("Id of the Targetlist: ")

        targetlist = adapter.get_targetlist(pipeline['id'], targetlist_id)
        targetlist = targetlist['content']
        if not targetlist:
            self.app.render('No targetlist exists with id: {0}. Set a valid id'.format(targetlist_id))
            return

        render_data = []

        if targetlist['target_list_type'] == 'DEVICES':
            targets = adapter.get_targetlist_targets(pipeline['id'], targetlist_id)
            targets = targets['content']['results']
            for target in targets:
                target_render = {
                    'Device Id': target['device_id'],
                    'Target List': targetlist_id,
                    'Created At': targetlist['created_at'],
                }
                render_data.append(target_render)
        if targetlist['target_list_type'] == 'GROUPS':
            devicegroup = adapter.get_targetlist_devicegroups(pipeline['id'], targetlist_id)
            devicegroup = devicegroup['content']['results']
            if not devicegroup:
                self.app.render('No device group present for Target list')
                return
            devicegroup = devicegroup[0]

            self.app.render('Fetching device group devices. This operation takes a few seconds based on the device count')

            devices = adapter.get_targetlist_devicegroup_devices(
                pipeline['id'],
                targetlist_id,
                devicegroup['id'])

            for device in devices:
                target_render = {
                    'Device Id': device['id'],
                    'Device Name': device['device_name'],
                    'Device Type': device['device_type'],
                    'Serial': device['serial'],
                }
                render_data.append(target_render)
        else:
            self.app.render("Unsupported device group type {0}".format(targetlist['target_list_type']))
            return

        self.app.render("Listing targetlist contents: {0} \n".format(targetlist_id))
        self.app.render(render_data, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")

    @ex(
        help='Remove a Target List',
        arguments=[
            (['-s', '--stage'],
             {'help': 'Stage to attach this list to',
              'action': 'store',
              'dest': 'stage',
              'default': None}),
            (['-t', '--targetlist-id'],
             {'help': 'Targetlist ID',
              'action': 'store',
              'dest': 'targetlist_id',
              'default': None})
        ]
    )
    def remove(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        auth_token = db.get_auth_token()
        adapter = PipelinesApiAdapter(environment, auth_token['auth_token'])

        stage_id = self.app.pargs.stage
        if not stage_id:
            self.app.render('Missing arguments! Provide a stage id to delete. ')
            stage_id = input("Id of the Stage: ")

        targetlist_id = self.app.pargs.targetlist_id
        if not targetlist_id:
            self.app.render('Missing arguments! Provide a targetlist id to delete. ')
            targetlist_id = input("Id of the Targetlist: ")

        pipeline = db.get_pipeline()
        if pipeline is None or pipeline.get('id') is None:
            self.app.log.debug('[pipeline-active] There is no active pipeline.')
            self.app.render('There is no active pipeline. Please set an active pipeline before deleting a target list')
            return

        adapter.delete_stage_target_list(stage_id, targetlist_id)

        self.app.render(f"Deleted Targetlist from stage Successfully! \n")
