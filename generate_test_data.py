import requests, random, time, json

url = 'http://localhost:32851/decety/'
num_images = 10

s = requests.Session()
image_ids = []

# login
s.post(url + 'dc-admin-p/', data={'login': 'admin', 'password': 'rqMhPnODHeam'})

valid_token = str(random.randint(0, 100000000))
invalid_token = str(random.randint(0, 100000000))
valid_shop_id = str(random.randint(0, 10000))
invalid_shop_id = str(random.randint(0, 10000))

# create valid token 
s.post(url + 'dc-admin-p/tokens', data={'v': 'create', 'token': valid_token, 'shop_id': valid_shop_id, 'description': '', 
	'exp_time': int(time.time() + 1e6)})

# create invalid token 
s.post(url + 'dc-admin-p/tokens', data={'v': 'create', 'token': invalid_token, 'shop_id': invalid_shop_id, 
	'description': 'expired test token', 'exp_time': int(time.time() - 1e6)})

# upload test images
for i in range(num_images):
	resp = s.post(url + 'upload', data={'token': valid_token}, files={'image': open('sample.jpg', 'rb')})
	image_id = json.loads(resp.text)['result']
	image_ids.append(image_id)


def create_item(token, item_id, color, size, description, type_):
	random.shuffle(image_ids)
	image_list = ','.join(image_ids[:random.randint(1, len(image_ids)-1)])
	data = {'token': token, 'id': item_id, 'color': color, 'size': size, 'description': description, 'type': type_, 'image_ids': image_list,
		'd1': random.uniform(1600, 1900),
		'd2': random.uniform(700, 1500),
		'd3': random.uniform(900, 1500),
		'd4': random.uniform(900, 1500),
		'd5': random.uniform(300, 500)}
	s.post(url + 'update', data=data)

create_item(valid_token, 'Mamalicious maternity jersey shorts', 'black', 'M', 'test description', 0)
create_item(valid_token, 'Mamalicious maternity jersey shorts', 'black', 'M', 'test description', 1)
create_item(valid_token, 'Mamalicious maternity jersey shorts', 'black', 'M', 'test description', 2)
create_item(valid_token, 'Mamalicious maternity jersey shorts', 'black', 'M', 'test description', 3)
create_item(valid_token, 'Mamalicious maternity jersey shorts', 'black', 'S', 'test description', 0)
create_item(valid_token, 'Mamalicious maternity jersey shorts', 'black', 'S', 'test description', 1)
create_item(valid_token, 'Mamalicious maternity jersey shorts', 'black', 'S', 'test description', 2)
create_item(valid_token, 'Mamalicious maternity jersey shorts', 'red', 'M', 'test description', 0)
create_item(valid_token, 'Mamalicious maternity jersey shorts', 'red', 'M', 'test description', 1)
create_item(valid_token, 'Mamalicious maternity jersey shorts', 'red', 'S', '', 0)
create_item(valid_token, 'Mamalicious maternity jersey shorts', 'red', 'S', '', 11)
create_item(valid_token, 'Lacoste croco sliders', 'black', 'M', 'test description', 3)
create_item(valid_token, 'Lacoste croco sliders', 'green', 'S', 'test description', 0)
create_item(valid_token, 'Lacoste croco sliders', 'green', 'S', 'test description', 1)
create_item(valid_token, 'empty', '', '', '', 0)
create_item(valid_token, 'only color', 'red', '', '', 0)
create_item(valid_token, 'only size', '', 'Z', '', 0)
create_item(valid_token, 'only description', '', '', 'description', 0)
create_item(valid_token, 'empty', '', '', '', 1)
create_item(valid_token, 'only color', 'red', '', '', 1)
create_item(valid_token, 'only size', '', 'Z', '', 1)
create_item(valid_token, 'only description', '', '', 'description', 1)