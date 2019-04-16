
from esper.main import EsperTest

def test_esper(tmp):
    with EsperTest() as app:
        res = app.run()
        print(res)
        raise Exception

def test_command1(tmp):
    argv = ['command1']
    with EsperTest(argv=argv) as app:
        app.run()
