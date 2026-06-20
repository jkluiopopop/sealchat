type TiptapCoreModule = typeof import('@tiptap/core');

export type PerformanceCommandType = 'delay' | 'pause';

export interface PerformanceCommandAttrs {
  command: PerformanceCommandType;
  value?: number | string | null;
}

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    performanceCommand: {
      insertPerformanceCommand: (attrs: PerformanceCommandAttrs) => ReturnType;
    };
  }
}

export const createPerformanceCommandExtension = ({
  Node,
  mergeAttributes,
}: Pick<TiptapCoreModule, 'Node' | 'mergeAttributes'>) => Node.create({
  name: 'performanceCommand',

  inline: true,
  group: 'inline',
  atom: true,
  selectable: true,
  draggable: false,

  addAttributes() {
    return {
      command: {
        default: 'delay',
        parseHTML: (element: HTMLElement) => element.getAttribute('data-performance-command') || 'delay',
        renderHTML: (attributes: PerformanceCommandAttrs) => ({
          'data-performance-command': String(attributes.command || 'delay'),
        }),
      },
      value: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-performance-value'),
        renderHTML: (attributes: PerformanceCommandAttrs) => {
          if (attributes.value == null || attributes.value === '') {
            return {};
          }
          return {
            'data-performance-value': String(attributes.value),
          };
        },
      },
    };
  },

  parseHTML() {
    return [{ tag: 'span[data-performance-command]' }];
  },

  renderHTML({ HTMLAttributes }) {
    const command = String(HTMLAttributes.command || 'command');
    return [
      'span',
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes, {
        class: 'tiptap-performance-command',
        contenteditable: 'false',
      }),
      `[${command}]`,
    ];
  },

  addCommands() {
    return {
      insertPerformanceCommand:
        (attrs: PerformanceCommandAttrs) =>
        ({ commands }) =>
          commands.insertContent({
            type: this.name,
            attrs: {
              command: attrs.command,
              value: attrs.value ?? null,
            },
          }),
    };
  },
});
