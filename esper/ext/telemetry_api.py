from logging import Logger

import requests


class TelemetryAPIError(Exception):
    """Exceptions related to calling Telemetry API"""
    pass


def get_telemetry_url(environment: str,
                      enterprise_id: str,
                      device_id: str,
                      category: str,
                      metric: str,
                      from_time: str,
                      to_time: str,
                      period: str,
                      statistic: str) -> str:
    """
    Build and return telemetry url for scapi endpoint
    :param environment:
    :param enterprise_id:
    :param device_id:
    :param category:
    :param metric:
    :param from_time:
    :param to_time:
    :param period:
    :param statistic:
    :return:
    """

    url = f'https://{environment}-api.shoonyacloud.com/api/graph/{category}/{metric}/?from_time={from_time}&' \
          f'to_time={to_time}&period={period}&statistic={statistic}&device_id={device_id}&enterprise_id=' \
          f'{enterprise_id}'

    return url

