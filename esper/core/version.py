from cement.utils.version import get_version as cement_get_version

VERSION = (0, 0, 14, 'final', 0)


def get_version(version=VERSION):
    return cement_get_version(version)
