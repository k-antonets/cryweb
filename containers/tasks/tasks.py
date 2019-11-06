from celery import Celery
import sys
sys.path.insert(1, '/cry_processor/')

from cry_processor import Crylauncher
from os import path
import subprocess

app = Celery("tasks", broker="redis://redis:6379/", backend="redis://redis:6379/")

app.conf.default_queue = 'cry_py'

@app.task
def cryprocess(run_mode, fi, fr, rr, meta, wd, th):
    hm, pr, ma, r, a, nu, mra, k, s, f = '', 1, '', 'do', False, '', False, 21, True, True
    od = path.join(wd, 'cry')
    od_file = path.join(wd, 'cry_result.zip')
    fi = path.join(wd, fi)
    fr = path.join(wd, fr)
    rr = path.join(wd, rr)
    if run_mode == 'proteins':
        fr, rr = '', ''
    else:
        fi = ''
    if meta:
        meta = True
    Crylauncher.LaunchProcessor(od, fi, hm, pr, th, ma, r, a, nu, mra, k, fr, rr, meta, s, f)

    subprocess.call("zip -r {0} {1}".format(od_file, od), shell=True)
    return wd