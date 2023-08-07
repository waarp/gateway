Authentification
################

L'authentification des requêtes REST se fait au moyen de l'authentification
`HTTP basique <https://tools.ietf.org/html/rfc7617>`_.

Ce schéma d'authentification ce fait au moyen de l'en-tête HTTP
:http:header:`Authorization`. Pour s'authentifier, le client doit :

1. Obtenir le login et le mot de passe de l'utilisateur
2. Construire l'identifiant de l'utilisateur en concaténant le login,
   un caractère deux-points (``:``) et le mot de passe
3. Encoder l'identifiant obtenu en Base64
4. Préfixer l'identifiant encodé par une déclaration du schéma d'authentification
   basique ("Basic")


Par exemple, si l'utilisateur entre comme nom ``Aladdin`` et comme mot de passe
"open sesame", alors le client doit envoyer l'en-tête suivant ::

    Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==



En cas d'échec de l'authentification d'une requête, le serveur répondra par
un code HTTP :http:statuscode:`401`.

Par défaut, Gateway ne possède qu'un seul utilisateur ``admin`` (mot de passe:
``admin_password``) avec tous les droits,afin de permettre la mise en place la
configuration initiale de Gateway. Pour des raisons de sécurité, il est
fortement recommandé lors de l'installation de Gateway de créer de nouveaux
utilisateurs avec des droits plus restreints, puis de supprimer cet utilisateur
``admin``.
