from pathlib import Path

from OpenSSL import crypto
from cement.utils import fs


def init_certs(app):
    certs_folder = app.config.get('esper', 'certs_folder')
    path = fs.abspath(certs_folder)

    # Check if path exists
    if not Path(path).exists():
        app.log.debug(f"[init_certs] Creating Certs folder!")
        fs.ensure_dir_exists(path)

    app.extend('certs_path', path)
    app.extend('local_key', fs.abspath(app.config.get('esper', 'local_key')))
    app.extend('local_cert', fs.abspath(app.config.get('esper', 'local_cert')))
    app.extend('device_cert', fs.abspath(app.config.get('esper', 'device_cert')))


def cleanup_certs(app):
    '''
    Remove old certificates
    :param app: Cement App instance
    :return:
    '''

    if Path(app.local_key).exists():
        Path(app.local_key).unlink()

    if Path(app.local_cert).exists():
        Path(app.local_cert).unlink()

    if Path(app.device_cert).exists():
        Path(app.device_cert).unlink()

    app.log.debug("[cleanup_certs] Existing certificates removed!")


def create_self_signed_cert_root(ca_cert, ca_key):
    '''
    Create Root Certificate for Self-signed certificates
    :param ca_cert: Path to Root Public Key
    :param ca_key: Path to Root Private Key
    :return: True
    '''
    key_pair = crypto.PKey()
    key_pair.generate_key(crypto.TYPE_RSA, 2048)

    cert = crypto.X509()

    cert.get_subject().C = "US"
    cert.get_subject().ST = "California"
    cert.get_subject().L = "Santa Clara"
    cert.get_subject().O = "Esper Inc."
    cert.get_subject().CN = "Esper Server Root Certificate"

    cert.set_serial_number(1000)
    cert.gmtime_adj_notBefore(0)
    cert.gmtime_adj_notAfter(24 * 60 * 60)
    cert.set_issuer(cert.get_subject())

    cert.set_pubkey(key_pair)
    cert.sign(key_pair, digest='sha256')

    with open(ca_cert, 'wb') as f:
        f.write(crypto.dump_certificate(crypto.FILETYPE_PEM, cert))

    with open(ca_key, 'wb') as f:
        f.write(crypto.dump_privatekey(crypto.FILETYPE_PEM, key_pair))

    return True


def create_signed_cert(ca_cert, ca_key, local_cert, local_key):
    '''
    Create self-signed certificates using Root CA certificates
    :param ca_cert:
    :param ca_key:
    :param local_cert:
    :param local_key:
    :return:
    '''

    with open(ca_cert, 'rb') as ca_cert_file:
        ca_cert = crypto.load_certificate(crypto.FILETYPE_PEM, ca_cert_file.read())

    with open(ca_key) as ca_key_file:
        ca_key = crypto.load_privatekey(crypto.FILETYPE_PEM, ca_key_file.read())

    key_pair = crypto.PKey()
    key_pair.generate_key(crypto.TYPE_RSA, 2048)

    # Raise a CSR
    cert_req = crypto.X509Req()

    cert_req.get_subject().C = "US"
    cert_req.get_subject().ST = "California"
    cert_req.get_subject().L = "Santa Clara"
    cert_req.get_subject().O = "Esper Inc."
    cert_req.get_subject().CN = "Esper Server Certificate"

    cert_req.set_pubkey(key_pair)
    cert_req.sign(key_pair, 'sha256')

    # Prepare a Certificate
    cert = crypto.X509()

    cert.gmtime_adj_notBefore(0)
    cert.gmtime_adj_notAfter(60 * 60)
    cert.set_issuer(ca_cert.get_subject())
    cert.set_subject(cert_req.get_subject())

    # Set PubKey as CSR Pubkey and Sign with CA key
    cert.set_pubkey(cert_req.get_pubkey())
    cert.sign(ca_key, 'sha256')

    # Dump Public Key
    with open(local_cert, 'wb') as f:
        f.write(crypto.dump_certificate(crypto.FILETYPE_PEM, cert))
        f.write(crypto.dump_certificate(crypto.FILETYPE_PEM, ca_cert))

    # Dump Private Key
    with open(local_key, 'wb') as f:
        f.write(crypto.dump_privatekey(crypto.FILETYPE_PEM, key_pair))

    return True


def create_self_signed_cert(local_cert, local_key):
    '''

    :param local_cert:
    :param local_key:
    :return: True
    '''
    key_pair = crypto.PKey()
    key_pair.generate_key(crypto.TYPE_RSA, 2048)

    cert = crypto.X509()

    cert.get_subject().C = "US"
    cert.get_subject().ST = "California"
    cert.get_subject().L = "Santa Clara"
    cert.get_subject().O = "Esper Inc."
    cert.get_subject().CN = "Esper Self-Signed Client Certificate"

    cert.set_serial_number(1000)
    cert.gmtime_adj_notBefore(0)
    cert.gmtime_adj_notAfter(24 * 60 * 60)
    cert.set_issuer(cert.get_subject())

    cert.set_pubkey(key_pair)
    cert.sign(key_pair, digest='sha256')

    with open(local_cert, 'wb') as f:
        f.write(crypto.dump_certificate(crypto.FILETYPE_PEM, cert))

    with open(local_key, 'wb') as f:
        f.write(crypto.dump_privatekey(crypto.FILETYPE_PEM, key_pair))

    return True


def save_device_certificate(file_path, cert_contents):
    with open(file_path, 'w') as f:
        f.write(cert_contents)
