import yaml

data = yaml.safe_load(open('./numerics.yml', 'r'))
vals = data['values']

used = set()

print('//nolint')
print('package irc')
print()
print('const (')

def print_item(idx, item, obsolete=None, **kwargs):
    if idx in used: return

    origin = item.get('origin', '')

    origin_name = kwargs.pop('origin', '')
    if origin_name and not origin or origin_name not in origin: return

    kwargs['obsolete'] = obsolete
    for k, v in kwargs.items():
        if item.get(k) != v: return

    # Mark seen
    used.add(idx)

    print('{} = "{}"'.format(item['name'], item['numeric']), end='')

    if origin and origin != origin_name:
        print(' // {}'.format(origin), end='')

    print()

def print_specific(**kwargs):
    for index, item in enumerate(vals):
        print_item(index, item, **kwargs)


print('// RFC1459')
print_specific(origin='RFC1459')
print()
print('// RFC2812')
print_specific(origin='RFC2812')
print()
print('// IRCv3')
print_specific(origin='IRCv3')
print()
print('// Other')
print_specific(name='RPL_ISUPPORT')
print()
print('// Ignored')
print('//')
print('// Anything not in an RFC has not been included because')
print('// there are way too many conflicts to deal with.')
print('/*')
for index, item in enumerate(vals):
    print_item(index, item)
print('//*/')
print()
print('// Obsolete')
print('/*')
for index, item in enumerate(vals):
    print_item(index, item, obsolete=True)
print('//*/')

print(')')
