import numpy as np
import pandas as pd

df = pd.read_excel('./data.xlsx')
data = df.values[:,1:]
for i in range(data.shape[1]):
	if np.isnan(data[0, i]):
		data[0, i] = data[0, i-1]
data = data.astype(np.float32).T
print(data)

# weights = np.array([1.0 / data[:,i].std() for i in range(data.shape[1])])
weights = 1.0 / np.mean(data, axis=0)
weights /= np.linalg.norm(weights)
print(repr(weights))
