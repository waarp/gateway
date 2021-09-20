package r66

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestSelfPushOK(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 push transfer", func(c C) {
			makeTransfer(c, ctx, true)

			Convey("Once the transfer has been processed", func(c C) {
				processTransfer(c, ctx)

				Convey("Then it should have executed all the tasks in order", func(c C) {
					serverMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")
					clientMsgShouldBe(c, "CLIENT | PUSH | PRE-TASKS[0] | OK")
					serverMsgShouldBe(c, "SERVER | PUSH | POST-TASKS[0] | OK")
					clientMsgShouldBe(c, "CLIENT | PUSH | POST-TASKS[0] | OK")
					serverMsgShouldBe(c, "SERVER END TRANSFER OK")
					clientMsgShouldBe(c, "CLIENT END TRANSFER")

					checkTransfersOK(c, ctx)
				})
			})
		})
	})
}

func TestSelfPullOK(t *testing.T) {
	SkipConvey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 pull transfer", func(c C) {
			makeTransfer(c, ctx, false)

			Convey("Once the transfer has been processed", func(c C) {
				processTransfer(c, ctx)

				Convey("Then it should have executed all the tasks in order", func(c C) {
					serverMsgShouldBe(c, "SERVER | PULL | PRE-TASKS[0] | OK")
					clientMsgShouldBe(c, "CLIENT | PULL | PRE-TASKS[0] | OK")
					clientMsgShouldBe(c, "CLIENT | PULL | POST-TASKS[0] | OK")
					serverMsgShouldBe(c, "SERVER | PULL | POST-TASKS[0] | OK")
					serverMsgShouldBe(c, "SERVER END TRANSFER OK")
					clientMsgShouldBe(c, "CLIENT END TRANSFER")

					checkTransfersOK(c, ctx)
				})
			})
		})
	})
}

func TestSelfPushClientPreTasksFail(t *testing.T) {
	SkipConvey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 push transfer", func(c C) {
			makeTransfer(c, ctx, true)

			Convey("Given an error during the client's pre-tasks", func(c C) {
				errMsg, removeFail := addClientFailure(c, ctx, model.ChainPre)

				Convey("Once the transfer has been processed", func(c C) {
					processTransfer(c, ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						serverMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PUSH | PRE-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PUSH | PRE-TASKS[1] | ERROR")
						clientMsgShouldBe(c, "CLIENT | PUSH | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PUSH | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER END TRANSFER ERROR")
						clientMsgShouldBe(c, "CLIENT END TRANSFER")

						cTrans := &model.Transfer{
							Step: types.StepPreTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   0,
							TaskNumber: 1,
						}

						sTrans := &model.Transfer{
							Step: types.StepPreTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   0,
							TaskNumber: 1,
						}

						checkTransfersErr(c, ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func(c C) {
							retryTransfer(c, ctx, removeFail)

							Convey("Once the transfer has been processed", func(c C) {
								processTransfer(c, ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									serverMsgShouldBe(c, "SERVER | PUSH | POST-TASKS[0] | OK")
									clientMsgShouldBe(c, "CLIENT | PUSH | POST-TASKS[0] | OK")
									serverMsgShouldBe(c, "SERVER END TRANSFER OK")
									clientMsgShouldBe(c, "CLIENT END TRANSFER")

									checkTransfersOK(c, ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPushServerPreTasksFail(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 push transfer", func(c C) {
			makeTransfer(c, ctx, true)

			Convey("Given an error during the server's pre-tasks", func(c C) {
				errMsg, removeFail := addServerFailure(c, ctx, model.ChainPre)

				Convey("Once the transfer has been processed", func(c C) {
					processTransfer(c, ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						serverMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[1] | ERROR")
						serverMsgShouldBe(c, "SERVER | PUSH | ERROR-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PUSH | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER END TRANSFER ERROR")
						clientMsgShouldBe(c, "CLIENT END TRANSFER")

						cTrans := &model.Transfer{
							Step: types.StepSetup,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   0,
							TaskNumber: 0,
						}

						sTrans := &model.Transfer{
							Step: types.StepPreTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   0,
							TaskNumber: 1,
						}

						checkTransfersErr(c, ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func(c C) {
							retryTransfer(c, ctx, removeFail)

							Convey("Once the transfer has been processed", func(c C) {
								processTransfer(c, ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									clientMsgShouldBe(c, "CLIENT | PUSH | PRE-TASKS[0] | OK")
									serverMsgShouldBe(c, "SERVER | PUSH | POST-TASKS[0] | OK")
									clientMsgShouldBe(c, "CLIENT | PUSH | POST-TASKS[0] | OK")
									serverMsgShouldBe(c, "SERVER END TRANSFER OK")
									clientMsgShouldBe(c, "CLIENT END TRANSFER")

									checkTransfersOK(c, ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPullClientPreTasksFail(t *testing.T) {
	SkipConvey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 pull transfer", func(c C) {
			makeTransfer(c, ctx, false)

			Convey("Given an error during the client's pre-tasks", func(c C) {
				errMsg, removeFail := addClientFailure(c, ctx, model.ChainPre)

				Convey("Once the transfer has been processed", func(c C) {
					processTransfer(c, ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						serverMsgShouldBe(c, "SERVER | PULL | PRE-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PULL | PRE-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PULL | PRE-TASKS[1] | ERROR")
						clientMsgShouldBe(c, "CLIENT | PULL | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PULL | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER END TRANSFER ERROR")
						clientMsgShouldBe(c, "CLIENT END TRANSFER")

						cTrans := &model.Transfer{
							Step: types.StepPreTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   0,
							TaskNumber: 1,
						}

						sTrans1 := &model.Transfer{
							Step: types.StepPreTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   0,
							TaskNumber: 1,
						}

						sTrans2 := &model.Transfer{
							Step: types.StepData,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 0,
						}

						sTrans3 := &model.Transfer{
							Step: types.StepData,
							Error: types.TransferError{
								Code:    types.TeUnknownRemote,
								Details: "Session closed",
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 0,
						}

						checkTransfersErr(c, ctx, cTrans, sTrans1, sTrans2, sTrans3)

						Convey("When retrying the transfer", func(c C) {
							retryTransfer(c, ctx, removeFail)

							Convey("Once the transfer has been processed", func(c C) {
								processTransfer(c, ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									clientMsgShouldBe(c, "CLIENT | PULL | POST-TASKS[0] | OK")
									serverMsgShouldBe(c, "SERVER | PULL | POST-TASKS[0] | OK")
									serverMsgShouldBe(c, "SERVER END TRANSFER OK")
									clientMsgShouldBe(c, "CLIENT END TRANSFER")

									checkTransfersOK(c, ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPullServerPreTasksFail(t *testing.T) {
	SkipConvey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 pull transfer", func(c C) {
			makeTransfer(c, ctx, false)

			Convey("Given an error during the server's pre-tasks", func(c C) {
				errMsg, removeFail := addServerFailure(c, ctx, model.ChainPre)

				Convey("Once the transfer has been processed", func(c C) {
					processTransfer(c, ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						serverMsgShouldBe(c, "SERVER | PULL | PRE-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PULL | PRE-TASKS[1] | ERROR")
						serverMsgShouldBe(c, "SERVER | PULL | ERROR-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PULL | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER END TRANSFER ERROR")
						clientMsgShouldBe(c, "CLIENT END TRANSFER")

						cTrans := &model.Transfer{
							Step: types.StepSetup,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   0,
							TaskNumber: 0,
						}

						sTrans := &model.Transfer{
							Step: types.StepPreTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   0,
							TaskNumber: 1,
						}

						checkTransfersErr(c, ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func(c C) {
							retryTransfer(c, ctx, removeFail)

							Convey("Once the transfer has been processed", func(c C) {
								processTransfer(c, ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									clientMsgShouldBe(c, "CLIENT | PULL | PRE-TASKS[0] | OK")
									clientMsgShouldBe(c, "CLIENT | PULL | POST-TASKS[0] | OK")
									serverMsgShouldBe(c, "SERVER | PULL | POST-TASKS[0] | OK")
									serverMsgShouldBe(c, "SERVER END TRANSFER OK")
									clientMsgShouldBe(c, "CLIENT END TRANSFER")

									checkTransfersOK(c, ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPushClientPostTasksFail(t *testing.T) {
	SkipConvey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 push transfer", func(c C) {
			makeTransfer(c, ctx, true)

			Convey("Given an error during the client's post-tasks", func(c C) {
				errMsg, removeFail := addClientFailure(c, ctx, model.ChainPost)

				Convey("Once the transfer has been processed", func(c C) {
					processTransfer(c, ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						serverMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PUSH | PRE-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PUSH | POST-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PUSH | POST-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PUSH | POST-TASKS[1] | ERROR")
						clientMsgShouldBe(c, "CLIENT | PUSH | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PUSH | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER END TRANSFER ERROR")
						clientMsgShouldBe(c, "CLIENT END TRANSFER")

						cTrans := &model.Transfer{
							Step: types.StepPostTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 1,
						}

						sTrans := &model.Transfer{
							Step: types.StepPostTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 1,
						}

						checkTransfersErr(c, ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func(c C) {
							retryTransfer(c, ctx, removeFail)

							Convey("Once the transfer has been processed", func(c C) {
								processTransfer(c, ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									serverMsgShouldBe(c, "SERVER END TRANSFER OK")
									clientMsgShouldBe(c, "CLIENT END TRANSFER")

									checkTransfersOK(c, ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPushServerPostTasksFail(t *testing.T) {
	Convey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 push transfer", func(c C) {
			makeTransfer(c, ctx, true)

			Convey("Given an error during the server's post-tasks", func(c C) {
				errMsg, removeFail := addServerFailure(c, ctx, model.ChainPost)

				Convey("Once the transfer has been processed", func(c C) {
					processTransfer(c, ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						serverMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PUSH | PRE-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PUSH | POST-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PUSH | POST-TASKS[1] | ERROR")
						serverMsgShouldBe(c, "SERVER | PUSH | ERROR-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PUSH | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER END TRANSFER ERROR")
						clientMsgShouldBe(c, "CLIENT END TRANSFER")

						cTrans := &model.Transfer{
							Step: types.StepData,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 0,
						}

						sTrans := &model.Transfer{
							Step: types.StepPostTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 1,
						}

						checkTransfersErr(c, ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func(c C) {
							retryTransfer(c, ctx, removeFail)

							Convey("Once the transfer has been processed", func(c C) {
								processTransfer(c, ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									clientMsgShouldBe(c, "CLIENT | PUSH | POST-TASKS[0] | OK")
									serverMsgShouldBe(c, "SERVER END TRANSFER OK")
									clientMsgShouldBe(c, "CLIENT END TRANSFER")

									checkTransfersOK(c, ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPullClientPostTasksFail(t *testing.T) {
	SkipConvey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 pull transfer", func(c C) {
			makeTransfer(c, ctx, false)

			Convey("Given an error during the client's post-tasks", func(c C) {
				errMsg, removeFail := addClientFailure(c, ctx, model.ChainPost)

				Convey("Once the transfer has been processed", func(c C) {
					processTransfer(c, ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						serverMsgShouldBe(c, "SERVER | PULL | PRE-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PULL | PRE-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PULL | POST-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PULL | POST-TASKS[1] | ERROR")
						clientMsgShouldBe(c, "CLIENT | PULL | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PULL | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER END TRANSFER ERROR")
						clientMsgShouldBe(c, "CLIENT END TRANSFER")

						cTrans := &model.Transfer{
							Step: types.StepPostTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 1,
						}

						sTrans := &model.Transfer{
							Step: types.StepData,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 0,
						}

						checkTransfersErr(c, ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func(c C) {
							retryTransfer(c, ctx, removeFail)

							Convey("Once the transfer has been processed", func(c C) {
								processTransfer(c, ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									serverMsgShouldBe(c, "SERVER | PULL | POST-TASKS[0] | OK")
									serverMsgShouldBe(c, "SERVER END TRANSFER OK")
									clientMsgShouldBe(c, "CLIENT END TRANSFER")

									checkTransfersOK(c, ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestSelfPullServerPostTasksFail(t *testing.T) {
	SkipConvey("Given a r66 service", t, func(c C) {
		ctx := initForSelfTransfer(c)

		Convey("Given a new r66 pull transfer", func(c C) {
			makeTransfer(c, ctx, false)

			Convey("Given an error during the server's post-tasks", func(c C) {
				errMsg, removeFail := addServerFailure(c, ctx, model.ChainPost)

				Convey("Once the transfer has been processed", func(c C) {
					processTransfer(c, ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						serverMsgShouldBe(c, "SERVER | PULL | PRE-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PULL | PRE-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PULL | POST-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PULL | POST-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER | PULL | POST-TASKS[1] | ERROR")
						serverMsgShouldBe(c, "SERVER | PULL | ERROR-TASKS[0] | OK")
						clientMsgShouldBe(c, "CLIENT | PULL | ERROR-TASKS[0] | OK")
						serverMsgShouldBe(c, "SERVER END TRANSFER ERROR")
						clientMsgShouldBe(c, "CLIENT END TRANSFER")

						cTrans := &model.Transfer{
							Status: types.StatusError,
							Step:   types.StepFinalization,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 0,
						}

						sTrans := &model.Transfer{
							Status: types.StatusError,
							Step:   types.StepPostTasks,
							Error: types.TransferError{
								Code:    types.TeExternalOperation,
								Details: errMsg,
							},
							Progress:   uint64(len(testFileContent)),
							TaskNumber: 1,
						}

						checkTransfersErr(c, ctx, cTrans, sTrans)

						Convey("When retrying the transfer", func(c C) {
							retryTransfer(c, ctx, removeFail)

							Convey("Once the transfer has been processed", func(c C) {
								processTransfer(c, ctx)

								Convey("Then it should have executed all the tasks in order", func(c C) {
									serverMsgShouldBe(c, "SERVER END TRANSFER OK")
									clientMsgShouldBe(c, "CLIENT END TRANSFER")

									checkTransfersOK(c, ctx)
								})
							})
						})
					})
				})
			})
		})
	})
}
