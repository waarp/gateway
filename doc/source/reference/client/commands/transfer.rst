Gestion des transferts
======================

.. program:: waarp-gateway

.. option:: transfer

   Commande de gestion des transferts en cours. Doit être suivi d'une commande
   spécifiant l'action souhaitée.

   .. option:: add

      Commande de création de transfert.

      .. option::  -f <FILE>, --file=<FILE>

         Spécifie le chemin du fichier à transférer.

         **ATTENTION**: le chemin doit être accessible depuis la racine de la
         gateway.

      .. option:: -s <SERVER_ID>, --server_id=<SERVER_ID>

         L'identifiant du partenaire distant avec lequel le transfert va être
         effectué.

      .. option:: -a <ACCOUNT_ID>, --account_id=<ACCOUNT_ID>

         L'identifiant du compte distant utilisé par la gateway pour d'identifier
         auprès du partenaire de transfert.

      .. option:: -r <RULE_ID>, --rule=<RULE_ID>

         L'identifiant de la règle utilisée pour le transfert.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 transfer add --file=path/to/file --server_id=1 --account_id=1 --rule=1