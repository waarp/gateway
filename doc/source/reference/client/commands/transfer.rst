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

   .. option:: add

      Commande de consultation de transfert. L'ID du transfert doit être fournit
      en argument de programme après la commande.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 transfer get 1

   .. option:: list

     Commande de listing et filtrage de multiples transferts.

      .. option:: -l <LIMIT>, --limit=<LIMIT>

         Limite le nombre de maximum de transferts autorisés dans la réponse.
         Fixé à 20 par défaut.

      .. option:: -o <OFFSET>, --offset=<OFFSET>

         Fixe le numéro du premier transfert renvoyé.

      .. option:: -s <SORT>, --sort=<SORT>

         Spécifie l'attribut selon lequel les serveurs seront triés. Les choix
         possibles sont: tri par date (``start``), par identifiant (``id``), par
         partenaire (``agent_id``) ou par règle (``rule_id``).

      .. option:: -d, --desc

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

      .. option:: --server_id=<SERVER_ID>

         Filtre les transferts avec l'identifiant de partenaire renseigné avec
         ce paramètre. Le paramètre peut être renseigné plusieurs fois pour
         filtrer plusieurs partenaires à la fois.

      .. option:: --account_id=<ACCOUNT_ID>

         Filtre les transferts avec l'identifiant de compte partenaire renseigné
         avec ce paramètre. Le paramètre peut être renseigné plusieurs fois pour
         filtrer plusieurs comptes à la fois.

      .. option:: --rule_id=<RULE_ID>

         Filtre les transferts avec l'identifiant de règle renseigné avec ce
         paramètre. Le paramètre peut être renseigné plusieurs fois pour filtrer
         plusieurs règles à la fois.

      .. option:: --status=<STATUS>

         Filtre les transferts ayant actuellement le statut renseigné avec ce
         paramètre. Le paramètre peut être renseigné plusieurs fois pour filtrer
         plusieurs statuts à la fois.

      .. option:: --start=<START>

         Filtre les transferts ultérieurs à la date renseignée avec ce paramètre.
         La date doit être renseignée en suivant le format standard ISO 8601 tel
         qu'il est décrit dans la `RFC3339 <https://www.ietf.org/rfc/rfc3339.txt>`_.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 transfer list -l 10 -o 5 -s id -d --server_id=1 --account_id=1 --rule_id=1 --status=PLANNED --start=2019-01-01T12:00:00+02:00

