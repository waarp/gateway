Lister les certificats
======================

.. http:get:: /api/partners/(string:partner)/certificates

   Renvoie une liste des certificats emplissant les critères donnés en paramètres
   de requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les certificats seront triés *(défaut: name+)*
   :type sort: [name+|name-]

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resjson array certificates: La liste des certificats demandés
   :resjsonarr string name: Le nom du certificat
   :resjsonarr string privateKey: La clé privée du certificat
   :resjsonarr string publicKey: La clé publique du certificat
   :resjsonarr string certificate: Le certificat de l'entité


   .. admonition:: Exemple de requête

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/partners/waarp_sftp/certificates?limit=5 HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   .. admonition:: Exemple de réponse

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 453

         {
           "certificates": [{
             "name": "certificat_waarp_1",
             "privateKey": "<clé privée 1>",
             "publicKey": "<clé publique 1>",
             "cert": "<certificat 1>"
           },{
             "name": "certificat_waarp_2",
             "privateKey": "<clé privée 2>",
             "publicKey": "<clé publique 2>",
             "cert": "<certificat 2>"
           }]
         }