Client en ligne de commande
###########################

.. program:: waarp-gateway

``waarp-gateway`` est l'application terminal permettant d'envoyer des commandes
à une instance waarp-gatewayd.


Paramètres de connection
========================

**Options**

.. option:: --remote ADDR, -r ADDR

   L'adresse de l'instance de gateway à interroger. Ce paramètre est requis.

.. option:: --user USER, -u USER

   Le nom de l'utilisateur pour authentifier la requête. Ce paramètre est requis.

**Variables d'environnement**

.. envvar::  WG_PASSWORD

   Le mot de passe de l'utilisateur. Si la variable d'environnement est vide,
   le mot de passe sera demandé dans l'invité de commande.


Commandes
=========

**Status du service**

.. option:: status

   Affiche le statut de tous les services de la gateway interrogée.

**Gestion des partenaires**

.. option:: partner

   Commande de gestion des partenaires de transfert. Doit être suivi d'une commande
   spécifiant l'action souhaitée.

   .. option:: create

      Commande de création de partenaire.

      .. option:: --name NAME, -n NAME

         Spécifie le nom du nouveau partenaire créé. Doit être unique.

      .. option:: --address ADDR, -a ADDR

         L'adresse du partenaire. Peut être une adresse IP ou une adresse Web résolue
         par DNS.

      .. option:: --port PORT, -p PORT

         Le port TCP utilisé par le partenaire pour les connections entrantes.

      .. option:: --type TYPE, -t TYPE

         Le type du partenaire spécifiant le protocole utilisé par celui-ci pour
         les transferts.

   .. option:: get PARTNER

      Commande de consultation de partenaire. Le nom du partenaire doit être fournit
      en argument de programme après la commande.

   .. option:: list

      Commande de listing et filtrage de multiples partenaires.

      .. option:: --limit LIMIT, -l LIMIT

         Limite le nombre de maximum partenaires autorisés dans la réponse. Fixé à
         20 par défaut.

      .. option:: --offset OFFSET, -o OFFSET

         Fixe le numéro du premier partenaire renvoyé.

      .. option:: --sort SORT, -s SORT

         Spécifie l'attribut selon lequel les partenaires seront triés. Les choix
         possibles sont: tri par nom (`name`), par adresse (`address`) ou par
         type (`type`).

      .. option:: --descending, -d

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

      .. option:: --type TYPE, -t TYPE

         Filtre les partenaires utilisant le type renseigné avec ce paramètre.
         Le paramètre peut être renseigné plusieurs fois pour filtrer plusieurs
         type à la fois.

      .. option:: --address ADDR, -a ADDR

         Filtre les partenaires ayant l'adresse renseignée avec ce paramètre.
         Le paramètre peut être renseigné plusieurs fois pour filtrer plusieurs
         adresses à la fois.

   .. option:: update PARTNER

      Commande de modification d'un partenaire existant. Le nom du partenaire
      doit être renseigné en argument de programme, après les options de commande.

      .. option:: --name NAME, -n NAME

         Spécifie le nom du nouveau partenaire créé. Doit être unique.

      .. option:: --address ADDR, -a ADDR

         L'adresse du partenaire. Peut être une adresse IP ou une adresse Web résolue
         par DNS.

      .. option:: --port PORT, -p PORT

         Le port TCP utilisé par le partenaire pour les connections entrantes.

      .. option:: --type TYPE, -t TYPE

         Le type du partenaire spécifiant le protocole utilisé par celui-ci pour
         les transferts.

   .. option:: delete PARTNER

      Commande de suppression de partenaire. Le nom du partenaire à supprimer doit
      être spécifié en argument de programme, après la commande.


**Gestion des comptes partenaires**

.. option:: account

   Commande de gestion des comptes partenaires. Doit être suivi d'une commande
   spécifiant l'action souhaitée.

   .. option:: --partner PARTNER, -p PARTNER

      Spécifie le nom du partenaire auquel le ou les comptes sont rattachés. Ce
      paramètre est requis.

   .. option:: create

      Commande de création de compte.

      .. option:: --username NAME, -n NAME

         Spécifie le nom d'utilisateur du nouveau compte créé. Doit être unique
         pour un partenaire donné.

      .. option:: --password PASS, -p PASS

         Le mot de passe du nouveau compte partenaire.

   .. option:: list

      Commande de listing de multiples comptes.

      .. option:: --limit LIMIT, -l LIMIT

         Limite le nombre de maximum comptes autorisés dans la réponse. Fixé à
         20 par défaut.

      .. option:: --offset OFFSET, -o OFFSET

         Fixe le numéro du premier compte renvoyé.

      .. option:: --sort SORT, -s SORT

         Spécifie l'attribut selon lequel les comptes seront triés. Les choix
         possibles sont: tri par nom (`name`).

      .. option:: --descending, -d

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

   .. option:: update ACCOUNT

      Commande de modification d'un compte existant. Le nom d'utilisateur du compte
      doit être renseigné en argument de programme, après les options de commande.

      .. option:: --username NAME, -n NAME

         Spécifie le nom d'utilisateur du nouveau compte créé. Doit être unique
         pour un partenaire donné.

      .. option:: --password PASS, -p PASS

         Le mot de passe du nouveau compte partenaire.

   .. option:: delete ACCOUNT

      Commande de suppression de compte. Le nom d'utilisateur du compte à supprimer doit
      être spécifié en argument de programme, après la commande.


**Gestion des certificats de compte**

.. option:: certificate

   Commande de gestion des certificats de compte. Chaque entrée comprend toute
   la chaîne de certification. Doit être suivi d'une commande spécifiant
   l'action souhaitée.

   .. option:: --partner PARTNER, -p PARTNER

      Spécifie le nom du partenaire auquel le ou les certificats sont rattachés. Ce
      paramètre est requis.

   .. option:: --account ACCOUNT, -a ACCOUNT

      Spécifie le nom du compte partenaire auquel le ou les certificats sont rattachés. Ce
      paramètre est requis.

   .. option:: create

      Commande de création de certificat.

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

   .. option:: get CERT

      Commande de suppression de certificat. Le nom du certificat à supprimer doit
      être spécifié en argument de programme, après la commande.

   .. option:: list

      Commande de listing de multiples certificats.

      .. option:: --limit LIMIT, -l LIMIT

         Limite le nombre de maximum certificats autorisés dans la réponse. Fixé à
         20 par défaut.

      .. option:: --offset OFFSET, -o OFFSET

         Fixe le numéro du premier certificat renvoyé.

      .. option:: --sort SORT, -s SORT

         Spécifie l'attribut selon lequel les certificats seront triés. Les choix
         possibles sont: tri par nom (`name`).

      .. option:: --descending, -d

         Si présent, les résultats seront triés par ordre décroissant au lieu de
         croissant par défaut.

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

   .. option:: delete CERT

      Commande de suppression de certificat. Le nom du certificat à supprimer doit
      être spécifié en argument de programme, après la commande.