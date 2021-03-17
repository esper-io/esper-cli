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
        self.headers['Authorization'] = 'Bearer {0}'.format(api_key)

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
