.. _reference-cli-client-transfers-resume:

######################
Reprendre un transfert
######################

.. program:: waarp-gateway transfer resume

Reprend le transfert donné si celui-ci est interrompu, en pause ou en erreur.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' transfer resume 1234
