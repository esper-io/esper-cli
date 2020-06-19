import esperclient as client
from esperclient.configuration import Configuration
import requests


class APIClient:
    def __init__(self, credential):
        self.config = Configuration()
        self.config.api_key['Authorization'] = credential["api_key"]
        self.config.api_key_prefix['Authorization'] = 'Bearer'
        self.config.host = f"https://{credential['environment']}-api.esper.cloud/api"

    def get_enterprise_api_client(self):
        return client.EnterpriseApi(client.ApiClient(self.config))

    def get_device_api_client(self):
        return client.DeviceApi(client.ApiClient(self.config))

    def get_application_api_client(self):
        return client.ApplicationApi(client.ApiClient(self.config))

    def get_command_api_client(self):
        return client.CommandsApi(client.ApiClient(self.config))

    def get_group_api_client(self):
        return client.DeviceGroupApi(client.ApiClient(self.config))

    def get_group_command_api_client(self):
        return client.GroupCommandsApi(client.ApiClient(self.config))

    def get_remoteadb_api_client(self):
        return client.DeviceApi(client.ApiClient(self.config))

    def get_token_api_client(self):
        return client.TokenApi(client.ApiClient(self.config))
