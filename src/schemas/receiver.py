from src.schemas.transmitter import TransmitterCreate, TransmitterResponse

# Receiver currently uses the same payload/response contract as transmitter,
# but is exposed through dedicated receiver repository/router modules.
ReceiverCreate = TransmitterCreate
ReceiverResponse = TransmitterResponse
