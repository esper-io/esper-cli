import esperclient as client
from esperclient.configuration import Configuration


class APIClient:
    def __init__(self, credential):
        self.config = Configuration()
        self.config.username = credential["username"]
        self.config.password = credential["password"]
        self.config.host = f"https://{credential['host']}-api.shoonyacloud.com/api"

    def get_device_api_client(self):
        return client.DeviceApi(client.ApiClient(self.config))

    def get_application_api_client(self):
        return client.ApplicationApi(client.ApiClient(self.config))

    def get_application_upload_api_client(self):
        return client.EnterpriseApplicationApi(client.ApiClient(self.config))

    def get_command_api_client(self):
        return client.CommandsApi(client.ApiClient(self.config))

    def get_command_api_request(self, command_name, command_args):
        return client.CommandRequest(command_args=command_args,
                                     command=command_name)  # CommandRequest | command name to fire
