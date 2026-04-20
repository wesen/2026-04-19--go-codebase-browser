import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/query';
import { indexApi } from './indexApi';
import { sourceApi } from './sourceApi';

export const store = configureStore({
  reducer: {
    [indexApi.reducerPath]: indexApi.reducer,
    [sourceApi.reducerPath]: sourceApi.reducer,
  },
  middleware: (getDefault) =>
    getDefault().concat(indexApi.middleware, sourceApi.middleware),
});

setupListeners(store.dispatch);

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
