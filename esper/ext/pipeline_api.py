import requests


class APIException(Exception):
    pass


def get_pipeline_url(environment: str,
                     enterprise_id: str,
                     pipeline_id: str = None) -> str:
    """
    Build and return pipeline url for scapi endpoint
    :param environment:
    :param enterprise_id:
    :param pipeline_id:
    :param stage_id:
    :param operation_id:
    :return: Url
    """

    url = f'https://{environment}-api.esper.cloud/api/v1/enterprise/{enterprise_id}/pipeline/'

    if pipeline_id:
        url += f'{pipeline_id}/'

    return url


def get_pipeline_execute_url(environment: str,
                             enterprise_id: str,
                             pipeline_id: str,
                             execute_id: str = None,
                             action: str = None) -> str:
    """
    Build and return pipeline Execute url for scapi endpoint
    :param environment:
    :param enterprise_id:
    :param pipeline_id:
    :param execute_id:
    :param action:
    :return: Url
    """

    url = f'https://{environment}-api.esper.cloud/api/v1/enterprise/{enterprise_id}/pipeline/{pipeline_id}/execute/'

    if execute_id:
        url = f"{url}{execute_id}/"

        if action:
            url = f"{url}{action}/"

    return url


def get_stage_url(environment: str,
                  enterprise_id: str,
                  pipeline_id: str = None,
                  stage_id: str = None) -> str:
    """
    Build and return pipeline url for scapi endpoint
    :param environment:
    :param enterprise_id:
    :param pipeline_id:
    :param stage_id:
    :param operation_id:
    :return: Url
    """

    url = get_pipeline_url(environment, enterprise_id, pipeline_id)

    url += "stage/"

    if stage_id:
        url += f'{stage_id}/'

    return url


def get_operation_url(environment: str,
                      enterprise_id: str,
                      pipeline_id: str = None,
                      stage_id: str = None,
                      operation_id: str = None) -> str:
    """
    Build and return pipeline url for scapi endpoint
    :param environment:
    :param enterprise_id:
    :param pipeline_id:
    :param stage_id:
    :param operation_id:
    :return: Url
    """

    url = get_stage_url(environment, enterprise_id, pipeline_id, stage_id)

    url += "operation/"

    if operation_id:
        url += f'{operation_id}/'

    return url


def get_group_command_url(environment: str,
                          enterprise_id: str,
                          group_id: str) -> str:
    """
    Build and return pipeline url for scapi endpoint
    :param environment:
    :param enterprise_id:
    :param group_id:
    :return: Url
    """
    url = f'https://{environment}-api.esper.cloud/api/enterprise/{enterprise_id}/devicegroup/{group_id}/command/'
    return url


def create_pipeline(url, api_key, pipeline_name, pipeline_desc=None, trigger=None):
    try:
        data = {"name": pipeline_name}
        if pipeline_desc:
            data["description"] = pipeline_desc

        if trigger:
            data["trigger"] = trigger

        response = requests.post(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            json=data
        )

    except Exception as exc:
        raise APIException(exc)

    return response


def create_stage(url, api_key, stage_name, stage_order, stage_desc=None):
    try:
        data = {"name": stage_name, "ordering": stage_order}
        if stage_desc:
            data["description"] = stage_desc

        response = requests.post(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            data=data
        )

    except Exception as exc:
        raise APIException(exc)

    return response


def create_operation(url, api_key, operation_name, operation_action, operation_desc=None, group_url=None):
    try:
        data = {
            "name": operation_name,
            "action": operation_action
        }
        if operation_desc:
            data["description"] = operation_desc

        if group_url:

            if operation_action == "APP_INSTALL":
                command = "INSTALL"
            if operation_action == "APP_UNINSTALL":
                command = "UNINSTALL"
            if operation_action == "REBOOT":
                command = "REBOOT"

            data["action_args"] = {
                "method": "POST",
                "url": group_url,
                "body": {
                    "command": command,
                },
                "headers": {
                    'Authorization': f'Bearer {api_key}'
                }
            }

        response = requests.post(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            json=data
        )

    except Exception as exc:
        raise APIException(exc)

    return response


def edit_pipeline(url, api_key, pipeline_name=None, pipeline_desc=None, trigger=None):
    try:
        data = {}
        if pipeline_name:
            data["name"] = pipeline_name
        if pipeline_desc:
            data["description"] = pipeline_desc

        response = requests.patch(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            json=data
        )

    except Exception as exc:
        raise APIException(exc)

    return response


def edit_stage(url, api_key, stage_name=None, stage_order=None, stage_desc=None):
    try:
        data = {}
        if stage_name:
            data["name"] = stage_name
        if stage_order:
            data["ordering"] = stage_order
        if stage_desc:
            data["description"] = stage_desc

        response = requests.patch(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            data=data
        )

    except Exception as exc:
        raise APIException(exc)

    return response


def edit_operation(url, api_key, operation_name=None, operation_action=None, operation_desc=None):
    try:
        data = {}
        if operation_name:
            data["name"] = operation_name
        if operation_action:
            data["action"] = operation_action
        if operation_desc:
            data["description"] = operation_desc

        response = requests.patch(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            data=data
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


def list_stages(url, api_key):
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


def fetch_stages(url, api_key):
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


def delete_api(url, api_key):
    try:
        response = requests.delete(
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


def execute_pipeline(url, api_key, data=None):
    try:
        response = requests.post(
            url,
            data=data,
            headers={
                'Authorization': f'Bearer {api_key}'
            }
        )
    except Exception as exc:
        raise APIException(exc)

    return response


def list_execute_pipeline(url, api_key, params=None):
    try:
        response = requests.get(
            url,
            params=params,
            headers={
                'Authorization': f'Bearer {api_key}'
            }
        )
    except Exception as exc:
        raise APIException(exc)

    return response
