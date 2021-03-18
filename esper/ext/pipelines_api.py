import json
import requests
import urllib.parse

class PipelinesAPIException(Exception):
    pass

class PipelinesApiAdapter():

    headers = {
        'User-Agent': 'esper-cli',
        'Content-Type': 'application/json',
    }

    timeout = 2.000
    success_response_codes = [ 200, 201, 202 ]

    def __init__(self, environment, api_key):
        self.base_url = 'https://{0}-api.esper.cloud/api/pipelines/v0'.format(environment)
        self.headers['Authorization'] = 'Token {0}'.format(api_key)


    def _call(self, method, path, data={}):
        error_message = None
        response = None
        request_url = self.base_url + path
        try:
            response = None
            if method == 'GET':
                response = requests.get(
                    request_url,
                    params  = data,
                    headers = self.headers,
                    timeout = self.timeout,
                )
            elif method == 'POST':
                response = requests.post(
                    request_url,
                    json = data,
                    headers = self.headers,
                    timeout = self.timeout,
                )
            elif method == 'PUT':
                response = requests.put(
                    request_url,
                    json = data,
                    headers = self.headers,
                    timeout = self.timeout,
                )
            elif method == 'DELETE':
                response = requests.delete(
                    request_url,
                    headers = self.headers,
                    timeout = self.timeout,
                )
        except requests.Timeout:
            error_message = 'Request timed out after {0} seconds'.format(
                self.timeout)
        except requests.ConnectionError:
            error_message = f'Unable to connect to SCAPI server'
        except requests.RequestException:
            error_message = 'Exception while connecting to SCAPI server'
        except Exception:
            error_message = 'Application unable to connect to SCAPI server'

        if error_message is not None:
            raise PipelinesAPIException(error_message)

        response_data = None
        if response.status_code in self.success_response_codes:
            try:
                response_data = json.loads(response.content)
            except Exception as exc:
                raise PipelinesAPIException(exc)
        else:
            error_message = 'Cannot get data from Pipelines url: {0} ' \
                'status_code: {1} response: {2}'.format(
                    request_url, response.status_code, response.content)
            raise PipelinesAPIException(error_message)

        return self._parse_response(response_data)

    def _parse_response(self, raw_response):
        return raw_response

    def create_pipeline(self, data):
        method = 'POST'
        path = '/pipelines/'

        return self._call(method, path, data)

    def get_pipeline(self, pipeline_id):
        method = 'GET'
        path = '/pipelines/{0}'.format(pipeline_id)

        return self._call(method, path)

    def get_pipelines(self):
        method = 'GET'
        path = '/pipelines/'

        return self._call(method, path)

    def delete_pipeline(self, pipeline_id):
        method = 'DELETE'
        path = '/pipelines/{0}'.format(pipeline_id)

        return self._call(method, path)

    def get_stages(self, pipeline_id):
        method = 'GET'
        path = '/pipelines/{0}/stages/'.format(pipeline_id)

        return self._call(method, path)

    def create_stage(self, pipeline_id, data):
        method = 'POST'
        path = '/pipelines/{0}/stages/'.format(pipeline_id)

        return self._call(method, path, data)

    def delete_stage(self, pipeline_id, stage_id):
        method = 'DELETE'
        path = '/pipelines/{0}/stages/{1}/'.format(pipeline_id, stage_id)

        return self._call(method, path)

    def get_targetlist(self, pipeline_id, targetlist_id):
        method = 'GET'
        path = '/pipelines/{0}/targetlists/{1}/'.format(pipeline_id, targetlist_id)

        return self._call(method, path)

    def create_targetlist(self, pipeline_id, data):
        method = 'POST'
        path = '/pipelines/{0}/targetlists/'.format(pipeline_id)

        return self._call(method, path, data)

    def add_targetlist_device(self, pipeline_id, targetlist_id, data):
        method = 'POST'
        path = '/pipelines/{0}/targetlists/{1}/targets/'.format(pipeline_id, targetlist_id)

        return self._call(method, path, data)

    def add_targetlist_device_group(self, pipeline_id, targetlist_id, data):
        method = 'POST'
        path = '/pipelines/{0}/targetlists/{1}/devicegroups/'.format(pipeline_id, targetlist_id)

        return self._call(method, path, data)

    def get_stage_target_lists(self, stage_id):
        method = 'GET'
        path = '/stages/{0}/targetlists/'.format(stage_id)

        return self._call(method, path)

    def add_stage_target_list(self, stage_id, targetlist_id):
        method = 'POST'
        path = '/stages/{0}/targetlists/'.format(stage_id)
        data = {
            'target_list_id': targetlist_id,
        }

        return self._call(method, path, data)

    def delete_stage_target_list(self, stage_id, targetlist_id):
        method = 'DELETE'
        path = '/stages/{0}/targetlists/{1}/'.format(stage_id, targetlist_id)

        return self._call(method, path)

    def get_targetlist_targets(self, pipeline_id, targetlist_id):
        method = 'GET'
        path = '/pipelines/{0}/targetlists/{1}/targets/'.format(pipeline_id, targetlist_id)

        return self._call(method, path)

    def get_targetlist_devicegroups(self, pipeline_id, targetlist_id):
        method = 'GET'
        path = '/pipelines/{0}/targetlists/{1}/devicegroups/'.format(pipeline_id, targetlist_id)

        return self._call(method, path)

    def get_targetlist_devicegroup_devices(self, pipeline_id, targetlist_id, devicegroup_id):
        method = 'GET'
        path = '/pipelines/{0}/targetlists/{1}/devicegroups/{2}/devices/'.format(pipeline_id, targetlist_id, devicegroup_id)
        data = {
            'offset': 0,
            'limit': 10,
        }

        result = []
        next_page = True
        while next_page:
            response = self._call(method, path, data)
            result.extend(response['content']['results'])

            if not response['content']['next']:
                next_page = False
            else:
                data['offset'] = data['limit'] * (data['offset'] + 1)

        return result
