import requests


class APIException(Exception):
    pass

def get_pipeline_url(environment: str,
                     enterprise_id: str,
                     pipeline_id: str = None,
                     stage_id: str = None,
                     operation_id: str = None) -> str:
    """
    Build and return telemetry url for scapi endpoint
    :param environment:
    :param enterprise_id:
    :param pipeline_id:
    :param stage_id:
    :param operation_id:
    :return: Url
    """

    url = f'https://{environment}-api.esper.cloud/api/enterprise/{enterprise_id}/pipeline/'

    if pipeline_id:
        url += f'{pipeline_id}/'

    if stage_id:
        url += f'stage/{stage_id}/'

    if operation_id:
        url += f'operation/{operation_id}'

    return url


def create_pipeline(url, api_key, pipeline_name, pipeline_desc=None):

    try:
        response = requests.post(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            data={
                "name": pipeline_name,
                "description": pipeline_desc
            }
        )

    except Exception as exc:
        raise APIException(exc)

    return response


def edit_pipeline(url, api_key, pipeline_name=None, pipeline_desc=None):
    try:
        response = requests.patch(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            data={
                "name": pipeline_name,
                "description": pipeline_desc
            }
        )

    except Exception as exc:
        raise APIException(exc)

    return response


def list_pipelines(url, api_key):
    try:
        response = requests.get(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            }
        )

    except Exception as exc:
        raise APIException(exc)

    return response


def fetch_pipelines(url, api_key):
    try:
        response = requests.get(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            }
        )

    except Exception as exc:
        raise APIException(exc)

    return response


def render_single_dict(data):
    render_list = []
    for k, v in data.items():
        render_list.append(
            {
                "TITLE": k,
                "DETAILS": v
            }
        )

    return render_list
