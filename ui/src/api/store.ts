import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/query';
import { indexApi } from './indexApi';
import { sourceApi } from './sourceApi';
import { docApi } from './docApi';
import { xrefApi } from './xrefApi';
import { conceptsApi } from './conceptsApi';

export const store = configureStore({
  reducer: {
    [indexApi.reducerPath]: indexApi.reducer,
    [sourceApi.reducerPath]: sourceApi.reducer,
    [docApi.reducerPath]: docApi.reducer,
    [xrefApi.reducerPath]: xrefApi.reducer,
    [conceptsApi.reducerPath]: conceptsApi.reducer,
  },
  middleware: (getDefault) =>
    getDefault().concat(
      indexApi.middleware,
      sourceApi.middleware,
      docApi.middleware,
      xrefApi.middleware,
      conceptsApi.middleware,
    ),
});

setupListeners(store.dispatch);

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
