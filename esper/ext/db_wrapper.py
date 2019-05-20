from tinydb import Query


class DBWrapper:

    def __init__(self, db):
        self.db = db

    def set_configure(self, configure_data):
        Configure = Query()

        if self.db.get(Configure.config.exists()):
            self.db.remove(Configure.config.exists())

        self.db.insert({'config': configure_data})

    def get_configure(self):
        Configure = Query()

        db_result = self.db.get(Configure.config.exists())

        configure = None
        if db_result:
            configure = db_result['config']

        return configure

    def get_enterprise_id(self):
        configure = self.get_configure()
        return configure.get('enterprise_id') if configure else None

    def set_application(self, application):
        Application = Query()

        if self.db.get(Application.application.exists()):
            self.db.remove(Application.application.exists())

        self.db.insert({'application': application})

    def get_application(self):
        Application = Query()

        db_result = self.db.get(Application.application.exists())

        application = None
        if db_result:
            application = db_result['application']

        return application

    def unset_application(self):
        Application = Query()
        self.db.remove(Application.application.exists())

    def set_device(self, device):
        Device = Query()

        if self.db.get(Device.device.exists()):
            self.db.remove(Device.device.exists())

        self.db.insert({'device': device})

    def get_device(self):
        Device = Query()

        db_result = self.db.get(Device.device.exists())

        device = None
        if db_result:
            device = db_result['device']

        return device

    def unset_device(self):
        Device = Query()
        self.db.remove(Device.device.exists())

    def set_group(self, group):
        Group = Query()

        if self.db.get(Group.group.exists()):
            self.db.remove(Group.group.exists())

        self.db.insert({'group': group})

    def get_group(self):
        Group = Query()

        db_result = self.db.get(Group.group.exists())

        group = None
        if db_result:
            group = db_result['group']

        return group

    def unset_group(self):
        Group = Query()
        self.db.remove(Group.group.exists())
