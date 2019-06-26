
import matplotlib.pyplot as plt
plt.style.use('seaborn-whitegrid')

import pandas as pd

opendata = pd.read_csv('test_data/open_auction.csv')

opendata.loc['auctions'].plot()