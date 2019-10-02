Gestion des partenaires distants
================================

.. program:: waarp-gateway

.. option:: partner

   Commande de gestion des partenaires de transfert. Doit être suivi d'une commande
   spécifiant l'action souhaitée.

   .. option:: add

      Commande de création de partenaire.

      .. option::  -n <NAME>, --name=<NAME>

         Spécifie le nom du nouveau partenaire créé. Doit être unique.

      .. option:: -p <PROTO>, --protocol=<PROTO>

         Le protocole utilisé par le partenaire.

      .. option:: --config=<CONF>

         La configuration du partenaire en format JSON. Contient les informations
         nécessaires pour se connecter au partenaire. Le contenu de la configuration
         pouvant varier en fonction du protocole utilisé, cette configuration
         est stockée en format JSON *raw*.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 partner add -n partner_sftp -p sftp --config={"address": "10.0.0.1", "port": 21}

   .. option:: list

      Commande de listing et filtrage de multiples partenaires.

      .. option:: -l <LIMIT>, --limit=<LIMIT>

         Limite le nombre de maximum partenaires autorisés dans la réponse. Fixé à
         20 par défaut.

      .. option:: -o <OFFSET>, --offset=<OFFSET>

         Fixe le numéro du premier partenaire renvoyé.

      .. option:: -s <SORT>, --sort=<SORT>

         Spécifie l'attribut selon lequel les partenaires seront triés. Les choix
         possibles sont: tri par nom (``name``) ou par protocole (``protocol``)

      .. option:: -d, --desc

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

      .. option:: -p <PROTO>, --protocol=<PROTO>

         Filtre les partenaires utilisant le protocole renseigné avec ce paramètre.
         Le paramètre peut être renseigné plusieurs fois pour filtrer plusieurs
         protocoles à la fois.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 partner list -l 10 -o 5 -s protocol -d -p sftp -p http

   .. option:: get <PARTNER_ID>

      Commande de consultation de partenaire. L'ID du partenaire doit être fournit
      en argument de programme après la commande.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 partner get 1

   .. option:: update <PARTNER_ID>

      Commande de modification d'un partenaire existant. L'ID du partenaire
      doit être renseigné en argument de programme.

      .. option::  -n <NAME>, --name=<NAME>

         Change le nom du partenaire. Doit être unique.

      .. option:: -p <PROTO>, --protocol=<PROTO>

         Change le protocole utilisé par le partenaire.

      .. option:: --config=<CONF>

         Change la configuration du partenaire.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 partner update 1 -n partner_http -p http --config={"address": "localhost", "port": 80}

   .. option:: delete <PARTNER_ID>

      Commande de suppression de partenaire. L'identifiant du partenaire à
      supprimer doit être spécifié en argument de programme.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 partner delete 1

