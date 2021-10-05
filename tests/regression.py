from testflows.core import *

from helpers.cluster import Cluster
from helpers.argparser import argparser

xfails = {
        # test_operator.py
        # "/regression/e2e.test_operator/test_003*": [(Error, "Hits tf timeout")],
        # "/regression/e2e.test_operator/test_006*": [(Error, "May hit timeout")],
        # "/regression/e2e.test_operator/test_009*": [(Fail, "May fail due to label changes")],
        # "/regression/e2e.test_operator/test_022*": [(Error, "Not supported yet. Timeout")],
        # "/regression/e2e.test_operator/test_024*": [(Fail, "Not supported yet")],

        # test_clickhouse.py
        "/regression/e2e.test_clickhouse/test_ch_001*": [(Fail, "Insert Quorum test need to refactoring")],

        # test_metrics_alerts.py
        "/regression/e2e.test_metrics_alerts/test_clickhouse_dns_errors*": [(Fail, "DNSError profile event behavior changed on 21.9, look https://github.com/ClickHouse/ClickHouse/issues/29624")]
}


@TestSuite
@XFails(xfails)
@ArgumentParser(argparser)
def regression(self, native):
    """ClickHouse Operator test regression suite.
    """
    def run_features():
        features = [
            "e2e.test_metrics_exporter",
            "e2e.test_metrics_alerts",
            "e2e.test_zookeeper",
            "e2e.test_backup_alerts",
            "e2e.test_operator",
            "e2e.test_clickhouse",
            "e2e.test_examples",
        ]
        for feature_name in features:
            Feature(run=load(feature_name, "test"))

    self.context.native = native
    if native:
        run_features()
    else:
        with Cluster():
            run_features()


if main():
    regression()
