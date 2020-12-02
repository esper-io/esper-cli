from cement import Controller, ex
from esperclient.rest import ApiException
from esper.controllers.enums import OutputFormat
from esper.ext.api_client import APIClient
from esper.ext.collection_api import eql_search
from esper.ext.db_wrapper import DBWrapper
from esper.ext.utils import validate_creds_exists, parse_error_message

class Collection(Controller):
    class Meta:
        label = 'eql'
        # text displayed at the top of --help output
        description = 'Commands to query devices in the enterpise.'

        # text displayed at the bottom of --help output
        epilog = 'Usage: espercli eql'

        stacked_type = 'nested'
        stacked_on = 'base'

    @ex(
        help='Search for devices satisfying the query',
        arguments=[
            (['eql_string'],
             {'help': 'EQL query',
              'action': 'store',
            }),
            (['-j', '--json'],
             {'help': 'Render result in Json format',
              'action': 'store_true',
              'dest': 'json'}),
        ]
    )
    def search(self):
        validate_creds_exists(self.app)
        db = DBWrapper(self.app.creds)
        environment = db.get_configure().get("environment")
        enterprise_id = db.get_enterprise_id()
        scapi_api_key = db.get_configure().get("api_key")

        eql_string = self.app.pargs.eql_string
        if not eql_string:
            self.app.render(f'No query specified. Use the -q, --query option to specify the query string\n')
            return
        
        try:
            ok, response = eql_search(environment, enterprise_id, scapi_api_key, eql_string)
            if not ok:
                self.app.log.debug(f"Response not OK. {response}")
                self.app.render(f"Failed to query for devices.\n")
                return
        except ApiException as e:
            self.app.log.error(f"Failed to fetch results.")
            self.app.render(f"ERROR: {parse_error_message(self.app, e)}\n")
            return
        self.app.render(f"Number of Devices: {response['total']}\n")

        devices = []
        
        if not self.app.pargs.json:
            label = {
                'device': "DEVICE",
                'group': "GROUP",
                'brand': "BRAND",
                'dpc': "DPC Version",
                'state': "STATE",
                'status': "STATUS"
            }
            
            for device in response['results']:
                devices.append(
                    {
                        label['device']: device.get('name'),
                        label['group']: device.get('group', {}).get('name'),
                        label['brand']: device.get('hardware', {}).get('brand'),
                        label['dpc']: device.get('dpc', {}).get('version'),
                        label['state']: device.get('state'),
                        label['status']: device.get('status'),
                    }
                )
            self.app.render(devices, format=OutputFormat.TABULATED.value, headers="keys", tablefmt="plain")
        else:
            devices = []
            for device in response['results']:
                devices.append(
                    {
                        'device': device.get('name'),
                        'group': device.get('group', {}).get('name'),
                        'brand': device.get('hardware', {}).get('brand'),
                        'dpc': device.get('dpc', {}).get('version'),
                        'state': device.get('state'),
                        'status': device.get('status'),
                    }
                )
            self.app.render(devices, format=OutputFormat.JSON.value)






        
        

        
