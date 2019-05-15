from cement.core.output import OutputHandler
from cement.ext.ext_json import JsonOutputHandler
from cement.ext.ext_tabulate import TabulateOutputHandler

from esper.controllers.enums import OutputFormat


class EsperOutputHandler(OutputHandler):
    class Meta:
        label = 'esper_output_handler'

    def render(self, data, **kw):
        format = None
        if kw.get('format'):
            format = kw.pop('format')

        if not format:
            return str(data)

        if OutputFormat.TABULATED.value == format:
            return TabulateOutputHandler().render(data, **kw)
        elif OutputFormat.JSON.value == format:
            json_handler = JsonOutputHandler()
            json_handler._setup(self.app)
            return json_handler.render(data, **kw)
        else:
            self.app.log.error('Invalid output format.')
            self.app.exit_code = 0
