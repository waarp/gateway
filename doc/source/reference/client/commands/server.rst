Gestion des serveurs locaux
===========================

.. program:: waarp-gateway

.. option:: server

   Commande de gestion des serveurs locaux de la gateway. Doit être suivi d'une
   commande spécifiant l'action souhaitée.

   Comme les serveurs locaux de la gateway font partie intégrante de celle-ci,
   cette commande sert en fait à gérer la configuration de la gateway.

   .. option:: add

      Commande de création de serveur.

      .. option::  -n <NAME>, --name=<NAME>

         Spécifie le nom du nouveau serveur créé. Doit être unique.

      .. option:: -p <PROTO>, --protocol=<PROTO>

         Le protocole utilisé par le serveur.

      .. option:: --config=<CONF>

         La configuration du serveur en format JSON. Contient les informations
         nécessaires pour lancer le serveur. Le contenu de la configuration
         pouvant varier en fonction du protocole utilisé, cette configuration
         est stockée en format JSON *raw*.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 server add -n server_sftp -p sftp --config={"address": "localhost", "port": 21}

   .. option:: list

      Commande de listing et filtrage de multiples serveur.

      .. option:: -l <LIMIT>, --limit=<LIMIT>

         Limite le nombre de maximum de serveurs autorisés dans la réponse. Fixé
         à 20 par défaut.

      .. option:: -o <OFFSET>, --offset=<OFFSET>

         Fixe le numéro du premier serveur renvoyé.

      .. option:: -s <SORT>, --sort=<SORT>

         Spécifie l'attribut selon lequel les serveurs seront triés. Les choix
         possibles sont: tri par nom (``name``) ou par protocole (``protocol``)

      .. option:: -d, --desc

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

      .. option:: -p <PROTO>, --protocol=<PROTO>

         Filtre les serveurs utilisant le protocole renseigné avec ce paramètre.
         Le paramètre peut être renseigné plusieurs fois pour filtrer plusieurs
         protocoles à la fois.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 server list -l 10 -o 5 -s protocol -d -p sftp -p http

   .. option:: get <SERVER_ID>

      Commande de consultation de serveur. L'ID du serveur doit être fournit
      en argument de programme après la commande.

   **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 server get 1

   .. option:: update <SERVER_ID>

      Commande de modification d'un serveur existant. L'ID du serveur doit être 
      renseigné en argument de programme.

      .. option::  -n <NAME>, --name=<NAME>

         Change le nom du serveur. Doit être unique.

      .. option:: -p <PROTO>, --protocol=<PROTO>

         Change le protocole utilisé par le serveur.

      .. option:: --config=<CONF>

         Change la configuration du serveur.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 server update 1 -n server_http -p http --config={"address": "localhost", "port": 80}

   .. option:: delete <SERVER_ID>

      Commande de suppression de serveur. Le nom du serveur à supprimer doit
      être spécifié en argument de programme.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 server delete 1