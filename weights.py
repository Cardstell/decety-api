import numpy as np

data = np.array([[1, 0, 0, 0, 0], [0, 1, 0, 0, 0], [0, 0, 1, 0, 0], [0, 0, 0, 1, 0], [0, 0, 0, 0, 10]])
weights = np.array([1.0 / data[:,i].std() for i in range(data.shape[1])])
print(repr(weights))
