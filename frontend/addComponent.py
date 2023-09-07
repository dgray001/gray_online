import os
import string

def addDwgComponent():
  name = input('New component name: ').lower().replace(' ', '_')
  filename = name.split('/')[-1]
  html_class_name = filename.replace('_', '-')
  ts_class_name = string.capwords(filename.replace('_', ' ')).replace(' ', '')
  folder_depth_string = '../' * len(name.split('/'))
  folder_string = f'src/components/{name}'
  newdir = os.path.join(os.path.dirname(__file__), folder_string)

  if os.path.exists(newdir):
      print(f'A component with name {name} already exists')
      quit()
  print(f'\nCreating new compoennt with name {name}')

  os.mkdir(newdir)

  f = open(os.path.join(newdir, f'{filename}.html'), 'w')
  f.write(f'<div id="example">Hello from {html_class_name}!</div>\n')
  f.close()

  f = open(os.path.join(newdir, f'{filename}.ts'), 'w')
  f.write(f"import {{DwgElement}} from '{folder_depth_string}dwg_element';\n")
  f.write('\n')
  f.write(f"import html from './{filename}.html';\n")
  f.write(f"import './{filename}.scss';\n")
  f.write('\n')
  f.write(f'export class Dwg{ts_class_name} extends DwgElement {{\n')
  f.write('  example: HTMLDivElement;\n')
  f.write('\n')
  f.write('  constructor() {\n')
  f.write('    super();\n')
  f.write('    this.htmlString = html;\n')
  f.write("    this.configureElement('example');\n")
  f.write('  }\n')
  f.write('\n')
  f.write('  protected override parsedCallback(): void {\n')
  f.write(f"    console.log('Dwg{ts_class_name} parsed!');\n")
  f.write('  }\n')
  f.write('}\n')
  f.write('\n')
  f.write(f"customElements.define('dwg-{html_class_name}', Dwg{ts_class_name});\n")
  f.close()

  f = open(os.path.join(newdir, f'{filename}.scss'), 'w')
  f.close()


def addNewPage():
  print('not implemented')


i = input('CLI to add a new component to the frontend:\n\n  1. DwgElement component\n  2. New page\n\n  > ')
print('')
if i == '1':
  addDwgComponent()
elif i == '2':
  addNewPage()
else:
  print('Unrecognized command')