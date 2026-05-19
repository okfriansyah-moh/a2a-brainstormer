import { writable } from "svelte/store";

export interface ModalOptions {
  title: string;
  body: string;
  confirmLabel: string;
  confirmDanger?: boolean;
  onConfirm: () => void;
}

export interface UIStoreState {
  modalOpen: boolean;
  modalTitle: string;
  modalBody: string;
  modalConfirmLabel: string;
  modalConfirmDanger: boolean;
  onModalConfirm: (() => void) | null;
}

const initialState: UIStoreState = {
  modalOpen: false,
  modalTitle: "",
  modalBody: "",
  modalConfirmLabel: "Confirm",
  modalConfirmDanger: false,
  onModalConfirm: null,
};

function createUIStore() {
  const { subscribe, update } = writable<UIStoreState>(initialState);

  return {
    subscribe,

    openModal(opts: ModalOptions) {
      update((s) => ({
        ...s,
        modalOpen: true,
        modalTitle: opts.title,
        modalBody: opts.body,
        modalConfirmLabel: opts.confirmLabel,
        modalConfirmDanger: opts.confirmDanger ?? false,
        onModalConfirm: opts.onConfirm,
      }));
    },

    closeModal() {
      update((s) => ({
        ...s,
        modalOpen: false,
        onModalConfirm: null,
      }));
    },
  };
}

export const uiStore = createUIStore();
