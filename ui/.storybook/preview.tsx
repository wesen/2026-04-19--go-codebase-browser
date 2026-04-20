import type { Preview } from '@storybook/react';
import '../src/packages/ui/src/theme/base.css';
import '../src/packages/ui/src/theme/dark.css';
import { widgetRootAttrs } from '../src/packages/ui/src/parts';

const preview: Preview = {
  parameters: {
    controls: { matchers: { color: /(background|color)$/i, date: /Date$/i } },
  },
  globalTypes: {
    theme: {
      name: 'Theme',
      description: 'Widget theme override',
      defaultValue: 'light',
      toolbar: {
        icon: 'paintbrush',
        items: [
          { value: 'light', title: 'Light' },
          { value: 'dark', title: 'Dark' },
          { value: 'unstyled', title: 'Unstyled' },
        ],
        dynamicTitle: true,
      },
    },
  },
  decorators: [
    (Story, context) => {
      const theme = context.globals.theme as string;
      if (theme === 'unstyled') {
        return (
          <div style={{ padding: 16, all: 'revert' }}>
            <Story />
          </div>
        );
      }
      return (
        <div {...widgetRootAttrs} data-theme={theme} style={{ padding: 16 }}>
          <Story />
        </div>
      );
    },
  ],
};

export default preview;
