Gestion des comptes locaux
==========================

.. program:: waarp-gateway

.. option:: access

   Commande de gestion des comptes locaux. Doit être suivi d'une commande
   spécifiant l'action souhaitée.

   Un compte local est toujours rattaché à un serveur local, et permet donc aux
   partenaires distants de s'authentifier auprès de la gateway lors d'un transfert.

   .. option:: add

      Commande de création de compte.

      .. option:: --server_id=<SERVER_ID>

         L'identifiant numérique du serveur auquel le nouveau compte sera
         rattaché.

      .. option:: -l <LOGIN>, --login=<LOGIN>

         Spécifie le login du nouveau compte créé. Doit être unique pour un
         serveur donné.

      .. option:: -p <PASS>, --password=<PASS>

         Le mot de passe du nouveau compte.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 access add -l access1 -p password1 --server_id=1

   .. option:: list

      Commande de listing de multiples comptes.

      .. option:: -l <LIMIT>, --limit=<LIMIT>

         Limite le nombre de maximum comptes autorisés dans la réponse. Fixé à
         20 par défaut.

      .. option:: -o <OFFSET>, --offset=<OFFSET>

         Fixe le numéro du premier compte renvoyé.

      .. option:: -s <SORT>, --sort=<SORT>

         Spécifie l'attribut selon lequel les comptes seront triés. Les choix
         possibles sont: tri par login (`login`).

      .. option:: --descending, -d

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

      .. option:: --server_id=<SERVER_ID>

         Filtre les comptes rattachés au serveur renseigné avec ce paramètre.
         Le paramètre peut être renseigné plusieurs fois pour filtrer plusieurs
         serveurs à la fois.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 access list -l 10 -o 5 -s login -d --server_id=1 --server_id=2

   .. option:: get <ACCESS_ID>

      Commande de consultation de compte. L'identifiant du compte souhaité doit
      être spécifié en argument de programme.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 access get 1

   .. option:: update <ACCESS_ID>

      Commande de modification d'un compte existant. L'identifiant numérique du
      compte doit être renseigné en argument de programme.

      .. option:: --server_id=<SERVER_ID>

         Change le serveur auquel le compte est rattaché.

      .. option:: -l <LOGIN>, --login=<LOGIN>

         Change le nom d'utilisateur du compte. Doit être unique pour un
         serveur donné.

      .. option:: -p <PASS>, --password=<PASS>

         Change le mot de passe du compte.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 access update 1 -l access2 -p password2 --server_id=2

   .. option:: delete <ACCESS_ID>

      Commande de suppression de compte. Le nom d'utilisateur du compte à
      supprimer doit être spécifié en argument de programme.

      **Exemple**

      .. code-block:: bash

         waarp-gateway -a http://user:password@localhost:8080 access delete 1