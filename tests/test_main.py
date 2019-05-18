from esper.main import EsperTest


def test_esper():
    # test esper without any subcommands or arguments
    with EsperTest() as app:
        app.run()
        assert app.exit_code == 0


def test_esper_debug():
    # test that debug mode is functional
    argv = ['--debug']
    with EsperTest(argv=argv) as app:
        app.run()
        assert app.debug is True
