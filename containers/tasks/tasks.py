from celery import Celery
import sys
import glob
sys.path.insert(1, '/cry_processor/')

from cry_processor import Crylauncher
from os import path
import os
import subprocess

app = Celery("tasks", broker="redis://redis:6379/", backend="redis://redis:6379/")

app.conf.default_queue = 'cry_py'

@app.task
def cryprocess(run_mode, fi, fr, rr, meta, wd, th):
    hm, pr, ma, r, a, nu, mra, k, s, f = '', 1, '', 'do', False, '', False, 21, True, True
    od = path.join(wd, 'cry')
    final_result_dir = path.join(wd, 'cry_processor')
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

    os.mkdir(final_result_dir)
    os.replace(glob.glob(path.join(od, 'raw_full_*'))[0], path.join(final_result_dir, 'full_toxins.fasta'))
    os.replace(glob.glob(path.join(od, 'proteins_domain_mapping_full_*'))[0], path.join(final_result_dir, 'full_toxins.bed'))
    os.replace(path.join(od, 'logs', 'cry_processor.txt'), path.join(final_result_dir, 'summary_log.txt'))
    os.replace(glob.glob(path.join(od, 'logs', 'diamond_matches_*'))[0], path.join(final_result_dir, 'diamond_classification.txt'))
    subprocess.call("pushd {2}; zip -r {0} {1}; popd".format(od_file, final_result_dir, wd), shell=True)
    os.rmdir(od)
    os.rmdir(final_result_dir)
    return wd