Gestion des certificats
=======================

.. program:: waarp-gateway

.. option:: certificate

   Commande de gestion des certificats. Doit être suivi d'une commande spécifiant
   l'action souhaitée.

   Une entrée peut contenir: la chaîne de certification complète, la clé privée
   et la clé publique de l'entité.

   .. option:: add

      Commande de création de certificat.

      .. option:: -t <TYPE>, --type=<TYPE>

         Le type de l'entité à laquelle appartient le certificat. Valeurs
         possibles: `local_agents` (serveur), `remote_agents` (partenaire),
         `local_accounts` (compte local) et `remote_accounts` (compte partenaire).

      .. option:: -o <OWNER>, --owner=<OWNER>

         L'identifiant numérique de l'entité à laquelle appartient le certificat.

      .. option:: -n <NAME>, --name=<NAME>

         Spécifie le nom du nouveau certificat créé. Doit être unique pour une
         entité donné.

      .. option:: --private_key=<PRIV_KEY>

         Le chemin vers le fichier contenant la clé privée de l'entité.

      .. option:: --public_key=<PUB_KEY>

         Le chemin vers le fichier contenant la clé publique de l'entité.

      .. option:: --certificate=<CERT>

         Le chemin vers le fichier contenant le certificat de l'entité, avec
         la chaîne de certification complète.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 certificate add -t local_agents -o 1 -n certificate_sftp --public_key=.ssh/key.pub --private_key=.ssh/key --certificate=.ssh/cert.pem

   .. option:: get <CERT_ID>

      Commande de suppression de certificat. L'identifiant du certificat à
      supprimer doit être spécifié en argument de programme.

     **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 certificate get 1

   .. option:: list

      Commande de listing de multiples certificats.

      .. option:: -l <LIMIT>, --limit=<LIMIT>

         Limite le nombre de maximum certificats autorisés dans la réponse. Fixé à
         20 par défaut.

      .. option:: -o <OFFSET>, --offset=<OFFSET>

         Fixe le numéro du premier certificat renvoyé.

      .. option:: -s <SORT>, --sort=<SORT>

         Spécifie l'attribut selon lequel les certificats seront triés. Les choix
         possibles sont: tri par nom (`name`).

      .. option:: --descending, -d

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

      .. option:: --access=<ACCESS_ID>

         Filtre les certificats appartenant au compte local renseigné avec
         ce paramètre. Le paramètre peut être renseigné plusieurs fois pour
         filtrer plusieurs comptes à la fois.

      .. option:: --account=<ACCOUNT_ID>

         Filtre les certificats appartenant au compte partenaire renseigné avec
         ce paramètre. Le paramètre peut être renseigné plusieurs fois pour
         filtrer plusieurs comptes à la fois.

      .. option:: --partner=<PARTNER_ID>

         Filtre les certificats appartenant au partenaire distant renseigné avec
         ce paramètre. Le paramètre peut être renseigné plusieurs fois pour
         filtrer plusieurs comptes à la fois.

      .. option:: --server=<SERVER_ID>

         Filtre les certificats appartenant au serveur local renseigné avec
         ce paramètre. Le paramètre peut être renseigné plusieurs fois pour
         filtrer plusieurs comptes à la fois.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 certificate list 1 -l 10 -o 5 -s name -d --access=1 --account=1 --partner=1 --server=1

   .. option:: update CERT

      Commande de modification d'un certificat existant. Le nom du certificat
      doit être renseigné en argument de programme, après les options de commande.

      .. option:: --name NAME, -n NAME

         Spécifie le nom du nouveau certificat créé. Doit être unique pour un compte donné.

      .. option:: --private_key PRIV_KEY

         La clé privée du certificat.

      .. option:: --public_key PUB_KEY

         La clé publique du certificat.

      .. option:: --private_cert PRIV_CERT

         Le certificat privé.

      .. option:: --public_cert PUB_CERT

         Le certificat public.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 certificate update 1 -t remote_agents -o 2 -n certificate_http --public_key=http/key.pub --private_key=http/key --certificate=http/cert.pem

   .. option:: delete CERT

      Commande de suppression de certificat. Le nom du certificat à supprimer doit
      être spécifié en argument de programme, après la commande.

   **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 certificate delete 1