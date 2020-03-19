import yaml

data = yaml.safe_load(open('./numerics.yml', 'r'))
vals = data['values']

used = set()

print('//nolint')
print('package irc')
print()
print('const (')


def print_item(idx, item, ircv3=False, obsolete=None, tablevel=1, **kwargs):
    if idx in used:
        return

    origin = item.get('origin', '')
    origin_name = kwargs.pop('origin', '')

    if ircv3:
        if ('ircv3.net' not in item.get('contact', '')
                and 'ircv3.net' not in item.get('information', '')):
            return
    elif origin_name and not origin or origin_name not in origin:
        return

    kwargs['obsolete'] = obsolete
    for k, v in kwargs.items():
        if item.get(k) != v:
            return

    # Mark seen
    used.add(idx)

    print('\t' * tablevel, end='')

    print('{} = "{}"'.format(item['name'], item['numeric']), end='')

    if origin and origin != origin_name:
        print(' // {}'.format(origin), end='')

    print()


def print_specific(**kwargs):
    for index, item in enumerate(vals):
        print_item(index, item, **kwargs)


print('\t// RFC1459')
print_specific(origin='RFC1459')
print()
print('\t// RFC1459 (Obsolete)')
print_specific(origin='RFC1459', obsolete=True)
print()
print('\t// RFC2812')
print_specific(origin='RFC2812')
print()
print('\t// RFC2812 (Obsolete)')
print_specific(origin='RFC2812', obsolete=True)
print()
print('\t// IRCv3')
print_specific(origin='IRCv3', ircv3=True)
print()
#print('\t// IRCv3 (obsolete)')
#print_specific(origin='IRCv3', ircv3=True, obsolete=True)
#print()
print('\t// Other')
print_specific(name='RPL_ISUPPORT')
print()
print('\t// Ignored')
print('\t//')
print('\t// Anything not in an RFC has not been included because')
print('\t// there are way too many conflicts to deal with.')
print('\t/*')
print_specific(tablevel=2)
print('\t//*/')
print()
print('\t// Obsolete')
print('\t/*')
print_specific(obsolete=True, tablevel=2)
print('\t//*/')

print(')')
