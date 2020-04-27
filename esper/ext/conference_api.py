import requests
from esper.controllers.enums import DeviceCommandEnum


class APIException(Exception):
    pass


def get_room_api_url():
    return 'https://api.daily.co/v1/rooms'


def create_room(name, expiry, api_key):
    url = get_room_api_url()

    # You can also use "name": name, if not hipaa compliant
    data = {
        "properties": {
            "exp": expiry,
            "enable_chat": True
        }
    }

    try:
        response = requests.post(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            json=data
        )

    except Exception as exc:
        raise APIException(exc)

    return response.ok, response.json()


def get_room_by_name(name, api_key):
    url = f"{get_room_api_url()}/{name}"
    data = {
        "privacy": "public"
    }

    try:
        response = requests.get(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            json=data
        )

    except Exception as exc:
        raise APIException(exc)

    return response.ok, response.json()


def get_demo_config(environment, api_key):
    url = f"https://{environment}-api.esper.cloud/api/demo/config/"

    try:
        response = requests.get(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            }
        )

    except Exception as exc:
        raise APIException(exc)

    return response.ok, response.json()


def send_conference_command(environment, enterprise_id, device_id, api_key, room_details):
    url = f"https://{environment}-api.esper.cloud/api/v0/enterprise/{enterprise_id}/command/"

    data = {
        "command_type": "DEVICE",
        "command": DeviceCommandEnum.INITIATE_CONFERENCE_CALL.name,
        "command_args": room_details,
        "devices": [device_id],
        "device_type": "all"
    }

    try:
        response = requests.post(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            },
            json=data
        )

    except Exception as exc:
        raise APIException(exc)

    return response.ok, response.json()


def get_command_status(environment, enterprise_id, api_key, command_id):
    url = f"https://{environment}-api.esper.cloud/api/v0/enterprise/{enterprise_id}/command/{command_id}/status/"

    try:
        response = requests.get(
            url,
            headers={
                'Authorization': f'Bearer {api_key}'
            }
        )

    except Exception as exc:
        raise APIException(exc)

    return response.ok, response.json()
