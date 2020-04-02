Gestion de l'historique
=======================

.. program:: waarp-gateway

.. option:: history

   Commande de consultation de l'historique de transfert. Doit être suivi d'une
   commande spécifiant l'action souhaitée.

   .. option:: get

      Commande de consultation d'une entrée de l'historique. L'ID se l'entrée
      doit être fournit en argument de programme après la commande.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 history get 1

   .. option:: list

      Commande de listing et filtrage de multiples entrée d'historique.

      .. option:: -l <LIMIT>, --limit=<LIMIT>

         Limite le nombre de maximum d'entrée autorisées dans la réponse.
         Fixé à 20 par défaut.

      .. option:: -o <OFFSET>, --offset=<OFFSET>

         Fixe le numéro de la première entrée renvoyée.

      .. option:: -s <SORT>, --sort=<SORT>

         Spécifie l'attribut selon lequel les entrées seront triées. Les choix
         possibles sont: tri par date (``start``), par identifiant (``id``), par
         source du transfert (``source``), par destination du transfert
         (``destination``) ou par règle (``rule``).

      .. option:: -d, --desc

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

      .. option:: --source=<SOURCE>

         Filtre les entrées selon la source de transfert renseignée avec
         ce paramètre. Le paramètre peut être renseigné plusieurs fois pour
         filtrer plusieurs sources à la fois.

      .. option:: --destination=<DESTINATION>

         Filtre les entrées selon la destination de transfert renseignée avec
         ce paramètre. Le paramètre peut être renseigné plusieurs fois pour
         filtrer plusieurs destinations à la fois.

      .. option:: --rule=<RULE>

         Filtre les entrées avec le nom de règle renseigné avec ce paramètre.
         Le paramètre peut être renseigné plusieurs fois pour filtrer plusieurs
         règles à la fois.

      .. option:: --status=<STATUS>

         Filtre les entrées ayant terminées le statut renseigné dans ce
         paramètre. Le paramètre peut être renseigné plusieurs fois pour filtrer
         plusieurs statuts à la fois.

      .. option:: --protocol=<PROTOCOL>

         Filtre les entrée ayant ayant été effectuées avec le protocole
         renseigné dans ce paramètre. Le paramètre peut être renseigné plusieurs
         fois pour filtrer plusieurs statuts à la fois.

      .. option:: --start=<START>

         Filtre les entrées ultérieures à la date renseignée avec ce paramètre.
         La date doit être renseignée en suivant le format standard ISO 8601 tel
         qu'il est décrit dans la `RFC3339 <https://www.ietf.org/rfc/rfc3339.txt>`_.

      .. option:: --stop=<STOP>

         Filtre les entrées antérieures à la date renseignée avec ce paramètre.
         La date doit être renseignée en suivant le format standard ISO 8601 tel
         qu'il est décrit dans la `RFC3339 <https://www.ietf.org/rfc/rfc3339.txt>`_.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 history list -l 10 -o 5 -s id -d --source=compte_sftp --destination=serveur_sftp --rule=règle_sftp --protocol=sftp --status=DONE --start=2019-01-01T12:00:00+02:00 --stop=2019-01-02T12:00:00+02:00


   .. option:: restart

      Commande de redémarrage de transfert échoué. L'ID du transfert doit être
      fournit en argument de programme après la commande. Seuls les transferts
      ayant échoué peuvent être redémarrés.

      .. option:: -d <DATE>, --date=<DATE>

         La date à laquelle le transfert redémarrera. Par défaut, le transfert
         redémarre immédiatement. La date doit être renseignée en suivant le
         format standard ISO 8601 tel qu'il est décrit dans la
         `RFC3339 <https://www.ietf.org/rfc/rfc3339.txt>`_.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 history restart 1