import os
import time
import subprocess
import threading

threadCount = 128

os.system("go build -o main")

p = subprocess.Popen("exec ./main", shell=True)
print('Server started')
time.sleep(0.5)

def upload():
    os.system('curl -F "token=4gvsoCKuWhNe" -F "image=@/home/home/1.jpg" http://localhost:32851/decety/upload && echo ""')

threads = []
for i in range(threadCount):
    threads.append(threading.Thread(target=upload))
    threads[-1].start()
for th in threads:
    th.join()

time.sleep(0.5)
p.kill()
print('Server stopped')