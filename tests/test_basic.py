from tests.bins import BASIC_CONFIG_CLIENT, BASIC_CONFIG_SERVER, GoBin

def test_run_server():
    svbin = GoBin("server", BASIC_CONFIG_SERVER)
    svbin.start()
    svbin.assert_ready_ok()
    svbin.stop()

def test_run_client():
    svbin = GoBin("server", BASIC_CONFIG_SERVER)
    svbin.start()
    svbin.assert_ready_ok()

    clbin = GoBin("client", BASIC_CONFIG_CLIENT)
    clbin.start()
    clbin.assert_ready_ok()

    clbin.stop()
    svbin.stop()
