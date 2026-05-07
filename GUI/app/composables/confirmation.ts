import { useConfirm } from '#imports'

export function useConfirmation() {
  const confirm = useConfirm()
  const { showSuccessMessage, showInfoMessage } = useMessages()

  // eslint-disable-next-line unused-imports/no-unused-vars
  function doNothing(id: any) {
  }

  function confirmDelete(idToDelete: any, acceptCallback: (id: any) => void, rejectCallback: (id: any) => void = doNothing) {
    confirm.require({
      message: 'Should this entry be deleted ?',
      header: 'Are you sure',
      icon: 'pi pi-info-circle',
      rejectLabel: 'Cancel',
      acceptLabel: 'Delete',
      rejectClass: 'p-button-secondary p-button-outlined',
      acceptClass: 'p-button-danger',
      accept: () => {
        showSuccessMessage('Action confirmed', `Entry with ID ${idToDelete} was deleted`)
        acceptCallback(idToDelete)
      },
      reject: () => {
        showInfoMessage('Action cancelled', 'No changes are processed')
        rejectCallback(idToDelete)
      },
    })
  }

  function confirmAction(acceptCallback: () => void, acceptMessage: string = 'Action confirmed', acceptMessageDetail: string = acceptMessage, header: string = 'Attention', message: string = 'Should proceed with this action ?') {
    confirm.require({
      message,
      header,
      icon: 'pi pi-info-circle',
      rejectLabel: 'Cancel',
      acceptLabel: 'Accept',
      rejectClass: 'p-button-secondary p-button-outlined',
      acceptClass: 'p-button-success',
      accept: () => {
        acceptCallback()
        showInfoMessage(acceptMessage, acceptMessageDetail)
      },
      reject: () => {
        showInfoMessage('Action cancelled')
      },
    })
  }

  /**
   * Two-step confirmation for destructive deletes that cascade across the
   * database. Used when removing a system (Transmitter / Receiver /
   * Transponder) since deletion also drops every derived row in calibration,
   * on-board losses, test profiles, downlink calibration data, and
   * modulation-index measurements that referenced that code.
   *
   *  - Step 1 warns about the cascade impact and asks the user to confirm.
   *  - Step 2 requires a final "Yes, delete permanently" click.
   *
   * The accept callback only runs after BOTH confirmations pass.
   */
  function confirmCriticalDelete(
    entityKind: string,
    entityLabel: string,
    acceptCallback: () => void,
    rejectCallback: () => void = () => { /* noop */ },
  ) {
    const cascadeWarning =
      `Deleting ${entityKind} "${entityLabel}" will also remove ALL related ` +
      `entries from every dependent table (specifications, on-board losses, ` +
      `calibration, downlink calibration data, modulation-index measurements, ` +
      `test profiles, etc.). This cannot be undone.`

    confirm.require({
      message: cascadeWarning,
      header: `Delete ${entityKind}?`,
      icon: 'pi pi-exclamation-triangle',
      rejectLabel: 'Cancel',
      acceptLabel: 'Continue',
      rejectClass: 'p-button-secondary p-button-outlined',
      acceptClass: 'p-button-warning',
      accept: () => {
        // Second confirmation — require explicit final acknowledgement.
        confirm.require({
          message:
            `Final confirmation: permanently delete ${entityKind} ` +
            `"${entityLabel}" and all derived data?`,
          header: 'Are you absolutely sure?',
          icon: 'pi pi-exclamation-triangle',
          rejectLabel: 'Cancel',
          acceptLabel: 'Yes, delete permanently',
          rejectClass: 'p-button-secondary p-button-outlined',
          acceptClass: 'p-button-danger',
          accept: () => {
            acceptCallback()
          },
          reject: () => {
            showInfoMessage('Action cancelled', 'No changes were made')
            rejectCallback()
          },
        })
      },
      reject: () => {
        showInfoMessage('Action cancelled', 'No changes were made')
        rejectCallback()
      },
    })
  }

  return { confirmDelete, confirmAction, confirmCriticalDelete }
}
