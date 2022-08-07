from tests.bins import GoBin

def test_run_server():
    svbin = GoBin("server")
    svbin.start()
    svbin.assert_ready_ok()
    svbin.stop()

def test_run_client():
    svbin = GoBin("server")
    svbin.start()
    svbin.assert_ready_ok()

    clbin = GoBin("client")
    clbin.start()
    clbin.assert_ready_ok()

    clbin.stop()
    svbin.stop()
