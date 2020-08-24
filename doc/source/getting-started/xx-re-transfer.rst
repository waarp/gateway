####################
Création d'un rebond
####################

.. code-block:: sh

   gw rule update recv receive \
      -p '/upload' \
      -s '{"type": "MOVERENAME", "args": {"path":"#OUTPATH#/#ORIGINALFILENAME#"}}' \
      -s '{"type": "TRANSFER", "args":{"file": "#OUTPATH#/#ORIGINALFILENAME#", "to":"local", "as":"toto", "rule":"send_sftp"}}'
